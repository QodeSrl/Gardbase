# Changes

Provisioned the infrastructure for the new support-chat feature: a dedicated DynamoDB table, IAM access for the API instance, and the environment wiring to enable it.

## What changed
- `dynamodb.tf`: new `aws_dynamodb_table.chat` (`<project>-chat-<env>`) with `pk`/`sk` keys, provisioned capacity (5 in production, 1 otherwise), server-side encryption, and a `ttl` attribute so ephemeral conversations/messages expire automatically.
- `ec2.tf`: granted the API IAM role access to the chat table (and its indexes), and passed `dynamo_chat_table`, `staff_chat_token`, and `chat_allowed_origins` into the instance `user_data` template.
- `user_data.sh`: export `DYNAMO_CHAT_TABLE`, `STAFF_CHAT_TOKEN`, and `CHAT_ALLOWED_ORIGINS` and forward them as `-e` env vars to the API container.
- `variables.tf`: added `staff_chat_token` (sensitive, default empty → staff chat disabled) and `chat_allowed_origins` (default `*`).
- `outputs.tf`: exposed the chat table name under `dynamodb_tables.chat`.

## Notes
- The API gates the whole chat feature on `DYNAMO_CHAT_TABLE` being set, so provisioning the table is what turns the feature on.
- Leaving `staff_chat_token` empty disables the staff side while the visitor side still works.
- Terraform was not run/validated in the sandbox (no toolchain); changes were verified by reading the existing table/IAM/user_data patterns and mirroring them.
