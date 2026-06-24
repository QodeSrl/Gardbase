package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	ddbTypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
)

// ChatSender identifies who authored a message in a support conversation.
type ChatSender string

const (
	ChatSenderVisitor ChatSender = "visitor"
	ChatSenderStaff   ChatSender = "staff"
)

// ChatConversation is the metadata record for a single support conversation
// between a website visitor and the support staff.
//
// DynamoDB layout (single-table, lives in DYNAMO_CHAT_TABLE):
//
//	Conversation meta: pk = "CONV#<conversationId>", sk = "META"
//	Message:           pk = "CONV#<conversationId>", sk = "MSG#<RFC3339Nano ts>#<msgId>"
//
// Messages are range-ordered by their SK so a Query returns them
// chronologically without an extra sort.
type ChatConversation struct {
	PK string `dynamodbav:"pk" json:"-"`
	SK string `dynamodbav:"sk" json:"-"`

	ConversationID string    `dynamodbav:"conversation_id" json:"conversationId"`
	VisitorName    string    `dynamodbav:"visitor_name,omitempty" json:"visitorName,omitempty"`
	CreatedAt      time.Time `dynamodbav:"created_at" json:"createdAt"`
	UpdatedAt      time.Time `dynamodbav:"updated_at" json:"updatedAt"`
	TTL            int64     `dynamodbav:"ttl,omitempty" json:"-"`
}

// ChatMessage is a single persisted message inside a conversation.
type ChatMessage struct {
	PK string `dynamodbav:"pk" json:"-"`
	SK string `dynamodbav:"sk" json:"-"`

	ConversationID string     `dynamodbav:"conversation_id" json:"conversationId"`
	MessageID      string     `dynamodbav:"message_id" json:"id"`
	Sender         ChatSender `dynamodbav:"sender" json:"sender"`
	Body           string     `dynamodbav:"body" json:"body"`
	CreatedAt      time.Time  `dynamodbav:"created_at" json:"createdAt"`
	TTL            int64      `dynamodbav:"ttl,omitempty" json:"-"`
}

func chatConversationPK(conversationID string) string {
	return "CONV#" + conversationID
}

func chatMessageSK(createdAt time.Time, messageID string) string {
	return fmt.Sprintf("MSG#%s#%s", createdAt.UTC().Format(time.RFC3339Nano), messageID)
}

// CreateChatConversation creates a conversation meta record. If conversationID
// is empty a new UUID is generated. The created conversation is returned.
func (d *DynamoClient) CreateChatConversation(ctx context.Context, conversationID string, visitorName string) (*ChatConversation, error) {
	if d.ChatTable == "" {
		return nil, fmt.Errorf("chat table is not configured")
	}
	if conversationID == "" {
		conversationID = uuid.NewString()
	}
	now := time.Now().UTC()
	conv := &ChatConversation{
		PK:             chatConversationPK(conversationID),
		SK:             "META",
		ConversationID: conversationID,
		VisitorName:    visitorName,
		CreatedAt:      now,
		UpdatedAt:      now,
		TTL:            now.Add(d.chatRetention()).Unix(),
	}
	item, err := attributevalue.MarshalMap(conv)
	if err != nil {
		return nil, err
	}
	_, err = d.Client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(d.ChatTable),
		Item:      item,
	})
	if err != nil {
		return nil, err
	}
	return conv, nil
}

// GetChatConversation returns the conversation meta record, or nil if it does
// not exist.
func (d *DynamoClient) GetChatConversation(ctx context.Context, conversationID string) (*ChatConversation, error) {
	if d.ChatTable == "" {
		return nil, fmt.Errorf("chat table is not configured")
	}
	out, err := d.Client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(d.ChatTable),
		Key: map[string]ddbTypes.AttributeValue{
			"pk": &ddbTypes.AttributeValueMemberS{Value: chatConversationPK(conversationID)},
			"sk": &ddbTypes.AttributeValueMemberS{Value: "META"},
		},
	})
	if err != nil {
		return nil, err
	}
	if out.Item == nil {
		return nil, nil
	}
	var conv ChatConversation
	if err := attributevalue.UnmarshalMap(out.Item, &conv); err != nil {
		return nil, err
	}
	return &conv, nil
}

// AppendChatMessage persists a message and bumps the conversation's updated_at.
// The fully populated message (with id and timestamp) is returned.
func (d *DynamoClient) AppendChatMessage(ctx context.Context, conversationID string, sender ChatSender, body string) (*ChatMessage, error) {
	if d.ChatTable == "" {
		return nil, fmt.Errorf("chat table is not configured")
	}
	now := time.Now().UTC()
	msg := &ChatMessage{
		PK:             chatConversationPK(conversationID),
		SK:             chatMessageSK(now, uuid.NewString()),
		ConversationID: conversationID,
		MessageID:      uuid.NewString(),
		Sender:         sender,
		Body:           body,
		CreatedAt:      now,
		TTL:            now.Add(d.chatRetention()).Unix(),
	}
	item, err := attributevalue.MarshalMap(msg)
	if err != nil {
		return nil, err
	}
	if _, err := d.Client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(d.ChatTable),
		Item:      item,
	}); err != nil {
		return nil, err
	}

	// Best-effort bump of the conversation's updated_at timestamp.
	_, _ = d.Client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(d.ChatTable),
		Key: map[string]ddbTypes.AttributeValue{
			"pk": &ddbTypes.AttributeValueMemberS{Value: chatConversationPK(conversationID)},
			"sk": &ddbTypes.AttributeValueMemberS{Value: "META"},
		},
		UpdateExpression: aws.String("SET updated_at = :u"),
		ExpressionAttributeValues: map[string]ddbTypes.AttributeValue{
			":u": &ddbTypes.AttributeValueMemberS{Value: now.Format(time.RFC3339Nano)},
		},
	})

	return msg, nil
}

// ListChatMessages returns the persisted messages for a conversation in
// chronological order, capped at limit (defaults to 100, max 500).
func (d *DynamoClient) ListChatMessages(ctx context.Context, conversationID string, limit int) ([]ChatMessage, error) {
	if d.ChatTable == "" {
		return nil, fmt.Errorf("chat table is not configured")
	}
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	out, err := d.Client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(d.ChatTable),
		KeyConditionExpression: aws.String("pk = :pk AND begins_with(sk, :prefix)"),
		ExpressionAttributeValues: map[string]ddbTypes.AttributeValue{
			":pk":     &ddbTypes.AttributeValueMemberS{Value: chatConversationPK(conversationID)},
			":prefix": &ddbTypes.AttributeValueMemberS{Value: "MSG#"},
		},
		ScanIndexForward: aws.Bool(true),
		Limit:            aws.Int32(int32(limit)),
	})
	if err != nil {
		return nil, err
	}
	messages := make([]ChatMessage, 0, len(out.Items))
	for _, item := range out.Items {
		var msg ChatMessage
		if err := attributevalue.UnmarshalMap(item, &msg); err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}
	return messages, nil
}

// chatRetention is how long chat records are kept before the DynamoDB TTL
// reaper removes them.
func (d *DynamoClient) chatRetention() time.Duration {
	if d.ChatRetention > 0 {
		return d.ChatRetention
	}
	return 90 * 24 * time.Hour
}
