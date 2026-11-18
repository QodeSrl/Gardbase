// An isolated Web Crypto demonstration, not a Gardbase API client or wire format.
export type DemoEnvelope = {
  key: CryptoKey;
  iv: Uint8Array<ArrayBuffer>;
  ciphertext: ArrayBuffer;
  encoded: string;
};

export async function encryptDemoRecord(plaintext: string): Promise<DemoEnvelope> {
  const key = await crypto.subtle.generateKey({ name: "AES-GCM", length: 256 }, false, [
    "encrypt",
    "decrypt",
  ]);
  const iv = crypto.getRandomValues(new Uint8Array(12));
  const ciphertext = await crypto.subtle.encrypt(
    { name: "AES-GCM", iv },
    key,
    new TextEncoder().encode(plaintext)
  );
  const encoded = btoa(String.fromCharCode(...new Uint8Array(ciphertext)));
  return { key, iv, ciphertext, encoded };
}

export async function decryptDemoRecord(envelope: DemoEnvelope): Promise<string> {
  const plaintext = await crypto.subtle.decrypt(
    { name: "AES-GCM", iv: envelope.iv },
    envelope.key,
    envelope.ciphertext
  );
  return new TextDecoder().decode(plaintext);
}
