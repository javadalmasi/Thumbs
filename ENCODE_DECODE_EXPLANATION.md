# Encoding and Decoding Mechanism in Thumbs

## Overview

The Thumbs service implements a secure encoding/decoding mechanism that transforms 11-character YouTube video IDs into 12-character encoded IDs using XOR encryption with a secret key. This provides an additional layer of obfuscation while maintaining reversibility.

## Encoding Process

### Input Requirements
- **Input**: 11-character YouTube ID using base64-url alphabet (e.g., `xpAWlQa4UQQ`)
- **Secret Key**: Exactly 16-character secret key (defined in environment variable `SECRET_KEY`)
- **Output**: 12-character encoded ID using base64-url alphabet

### Step-by-Step Process

1. **Validation**: The input ID is validated to ensure it's exactly 11 characters and contains only valid base64-url characters (A-Z, a-z, 0-9, -, _)

2. **Base64-url Decoding**: The 11-character ID is decoded from base64-url encoding:
   - Since 11 characters of base64 represent 66 bits of data (11 × 6 = 66 bits), proper padding is added
   - If the input length mod 4 is 2, add `==` padding
   - If the input length mod 4 is 3, add `=` padding

3. **Key Derivation**: A 72-bit key is derived from the secret key and a salt:
   - Salt used: `"yt-11-to-12-salt-v1"`
   - The secret key and salt are concatenated and hashed using SHA256
   - The first 9 bytes (72 bits) of the hash are used as the derived key

4. **XOR Operation**: 
   - The decoded 66-bit value is shifted to the least significant bits of a 72-bit space
   - The XOR operation is performed: `M = N XOR S` where:
     - `N` = 66-bit input value in 72-bit space
     - `S` = 72-bit derived key
     - `M` = 72-bit result

5. **Base64-url Encoding**: 
   - The 72-bit result (`M`) is encoded back to base64-url
   - The first 12 characters are taken (9 bytes need exactly 12 base64-url characters)

## Example Encoding

```
Input ID:             xpAWlQa4UQQ (11 chars)
Environment Variable: SECRET_KEY=1234567890123456 (16 chars)
Encoded ID:           QKRqyXxmI1QY (12 chars)
```

## Decoding Process

### Input Requirements
- **Input**: 12-character encoded ID using base64-url alphabet
- **Secret Key**: Same 16-character secret key from environment variable `SECRET_KEY` used for encoding

### Step-by-Step Process

1. **Validation**: The input ID is validated to ensure it's exactly 12 characters and contains only valid base64-url characters

2. **Base64-url Decoding**: The 12-character encoded ID is decoded from base64-url encoding directly (12 chars = 9 bytes = 72 bits)

3. **Key Derivation**: The same 72-bit key is derived using the same salt and secret key

4. **XOR Operation**:
   - The XOR operation is performed to reverse the encoding: `N = M XOR S` where:
     - `M` = 72-bit encoded value
     - `S` = 72-bit derived key (same as in encoding)
     - `N` = 72-bit original value

5. **66-bit Masking**:
   - The result is masked to 66 bits: `N66 = N AND ((2^66) - 1)`
   - This ensures only the original 66 bits are preserved

6. **Base64-url Encoding**:
   - The 66-bit value is encoded to base64-url
   - The first 11 characters are taken to form the original 11-character ID

## Security Features

- **Deterministic**: The same input and secret key will always produce the same encoded output
- **Reversible**: The process can be perfectly reversed with the correct secret key
- **Key-based**: Without the correct 16-character secret key, decoding is computationally infeasible
- **Salted**: Uses a fixed salt to prevent rainbow table attacks
- **Obfuscated**: Transforms 11-character IDs to 12-character IDs, making them harder to guess

## Code Implementation

The encoding and decoding functions are implemented in the `internal/paths/vi.go` file:

```go
// Encode converts an 11-character YouTube ID to a 12-character encoded ID
func Encode(id11, secret string) (string, error) {
    // Implementation details as described above
}

// Decode converts a 12-character encoded ID back to an 11-character YouTube ID
func Decode(id12, secret string) (string, error) {
    // Implementation details as described above
}
```

## Requirements

- The `SECRET_KEY` environment variable must be set with exactly 16 characters
- Both input IDs must use only base64-url alphabet characters
- The encoding/decoding process requires the same secret key in both directions