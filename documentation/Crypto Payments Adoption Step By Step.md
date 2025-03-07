# Integrating TRC-20 USDT Payments into **hvmnd-api** and **hvmnd-web-app**

This guide explains how to add TRC-20 USDT deposit functionality to the **hvmnd** platform. We will use the TRON blockchain (via **TronGrid** free API) to generate deposit addresses per user, monitor incoming USDT transactions with 6 confirmation blocks, update user balances, and log payments. The design will also be extensible for future ERC-20 (Ethereum) and BEP-20 (BSC) tokens. Each step is detailed with code snippets and best practices for security and efficiency.

## Blockchain Provider: Setting up TronGrid for TRC-20 USDT

**TronGrid** will serve as our blockchain API for interacting with the TRON network. TronGrid provides full-node access through HTTP endpoints, which we’ll use to query transactions and events for USDT (TRC-20) transfers. 

- **TronGrid Endpoint**: We will use the TronGrid **v1 API** at `https://api.trongrid.io`. For example, TronGrid provides an endpoint to fetch TRC-20 token transfers for an address:  
  ```http
  GET https://api.trongrid.io/v1/accounts/{address}/transactions/trc20
  ``` 
  This returns TRC-20 transfers involving the given address. We can filter specifically for USDT by adding the contract address of USDT on Tron.

- **USDT TRC-20 Contract**: On Tron mainnet, USDT’s TRC-20 contract address is `TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t`. We will use this to identify USDT transfers. TronGrid’s account query can include a `contract_address` parameter to limit results to this token.

- **API Keys & Rate Limits**: The free tier of TronGrid should suffice for moderate usage. Sign up on TronGrid to obtain an API key if required, and configure the requests to include this key (e.g., as a header or URL parameter) for higher rate limits. The free tier supports a certain number of requests per second—batching calls (discussed later) will help stay within limits.

- **TronGrid in Code**: We can interact with TronGrid either by direct HTTP calls (e.g., using Go’s `net/http` or a library) or via TronWeb/TronGrid SDK in Node.js. In our Go backend, we’ll likely use direct HTTP requests to TronGrid’s REST API for simplicity. Ensure to use the **solidity node** API for confirmed data (TronGrid’s `api.trongrid.io` automatically serves confirmed data by default).

## Wallet Generation & Security (TRC-20 Deposit Addresses)

Each user needs a unique Tron **deposit address** to receive USDT. We will generate a Tron keypair for each user and securely store it.

### Generating Tron Wallet Addresses

Tron uses an Ethereum-like key scheme (secp256k1 ECDSA). We can generate a private key and derive the Tron address (which starts with **T**). One convenient way is using Tron’s official library **TronWeb** in a Node.js script or via an API:

```javascript
// Example using TronWeb (Node.js)
const TronWeb = require("tronweb");
const tronWeb = new TronWeb({
  fullHost: "https://api.trongrid.io"  // TronGrid full node
});
// Generate a new account (async)
const account = await tronWeb.createAccount();
console.log(account.address.base58, account.privateKey);
```

This yields an object with a new address and private key ([node.js - How to create a tron wallet with nodejs? - Stack Overflow](https://stackoverflow.com/questions/66651807/how-to-create-a-tron-wallet-with-nodejs#:~:text=const%20wallet%20%3D%20await%20tronWeb,log%28wallet)) ([node.js - How to create a tron wallet with nodejs? - Stack Overflow](https://stackoverflow.com/questions/66651807/how-to-create-a-tron-wallet-with-nodejs#:~:text=address%3A%20,41D3737C4D6B5105692B01409738D29CD796876602%27)). For example:

```json
{
  "address": {
    "base58": "TVFFvcUB6CWLFh45n28Ve1XRmu1NYSKS34",
    "hex": "41D3737C4D6B5105692B01409738D29CD796876602"
  },
  "privateKey": "D526E0AB73B3...914DD"
}
``` 

*Alternatively*, since our backend is in Go, we can use a Tron SDK or an Ethereum library to generate keys. The process in Go would be: generate 32-byte random private key, derive the public key (secp256k1), compute the address (last 20 bytes of keccak-256 hash of public key, with Tron’s 0x41 prefix), and encode in Base58Check ([Universal Private keys Tron Calculators address generator TRX](https://secretscan.org/PrivateKeyTron#:~:text=Universal%20Private%20keys%20Tron%20Calculators,x%20%2B%20y%20%C2%B7%202)). For simplicity, using TronWeb’s `createAccount()` via a small Node script or using an existing Tron Go library (like `github.com/okx/go-wallet-sdk/coins/tron` or similar) is recommended to avoid errors.

### Secure Storage of Private Keys

Security of private keys is paramount. We will **encrypt private keys** before storing them in PostgreSQL:

- **Encryption Method**: Use strong symmetric encryption (e.g., AES-256-GCM) with a server-held secret. The secret (encryption key) is set via environment variable and not stored in the code or database. For example, in Go:

  ```go
  import (
      "crypto/aes"
      "crypto/cipher"
      "crypto/rand"
  )
  // encryptPrivateKey encrypts a hex private key using AES-GCM
  func encryptPrivateKey(hexKey string, passphrase []byte) (string, error) {
      block, _ := aes.NewCipher(passphrase)
      aesGCM, _ := cipher.NewGCM(block)
      nonce := make([]byte, aesGCM.NonceSize())
      rand.Read(nonce)
      ciphertext := aesGCM.Seal(nonce, nonce, []byte(hexKey), nil)
      return hex.EncodeToString(ciphertext), nil
  }
  ```
  *(Error handling omitted for brevity.)*

- **Database Schema**: We will create a new table for crypto wallets, or extend the existing `webapp_users` table:
  - **Option 1**: **`user_wallets` table** (recommended for multi-chain support). Fields: `id (PK)`, `user_id (FK to users)`, `chain` (e.g. 'TRON'), `token` (e.g. 'USDT'), `address` (Tron address base58), `encrypted_private_key`, `created_at`.
  - **Option 2**: Add columns to `webapp_users`: e.g. `tron_deposit_address` and `tron_privkey_encrypted`. This is simpler but less extensible for additional chains.

  We will proceed with the **`user_wallets`** table for flexibility.

**PostgreSQL Schema Update (Wallet Storage)**: Example migration SQL for the new table:
```sql
CREATE TABLE public.crypto_deposit_addresses (
	id serial4 NOT NULL,
	user_id int4 NULL,
	network_id int4 NULL,
	address varchar(255) NOT NULL,
	is_used bool DEFAULT false NULL,
	created_at timestamptz DEFAULT CURRENT_TIMESTAMP NULL,
	used_at timestamptz NULL,
	CONSTRAINT crypto_deposit_addresses_pkey PRIMARY KEY (id),
	CONSTRAINT crypto_deposit_addresses_network_id_fkey FOREIGN KEY (network_id) REFERENCES public.crypto_networks(id) ON DELETE CASCADE,
	CONSTRAINT crypto_deposit_addresses_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.webapp_users(id) ON DELETE CASCADE
);
```
When a new user registers (or when they first request a crypto deposit address), generate a Tron address, encrypt the private key, and insert a record into `user_wallets` (with `chain='TRON'` and `token='USDT'`). Also consider generating this on-demand to avoid unused wallets – e.g., provide a “Generate Deposit Address” button in the UI that calls the API to create the address if not already present.

### Wallet Generation in Backend (hvmnd-api)

Implement a service or function in **hvmnd-api** for wallet creation. For example, add a new handler in `handlers/payments.go` or `handlers/webapp_users.go`:
```go
// Pseudocode for creating a TRON USDT wallet for a user
func CreateCryptoWallet(w http.ResponseWriter, r *http.Request) {
    userID := auth.GetUserID(r)  // assume we have the user’s ID from auth
    // 1. Generate Tron keypair (could call external script or use a library)
    address, privKey := tron.GenerateAddress()  // pseudocode
    // 2. Encrypt the private key
    encKey, err := encryptPrivateKey(privKey, []byte(os.Getenv("ENC_KEY")))
    // 3. Store in DB
    _, err = db.Exec(`INSERT INTO user_wallets(user_id, chain, token, address, encrypted_privkey) 
                      VALUES($1,$2,$3,$4,$5)`, userID, "TRON", "USDT", address, encKey)
    // 4. Return the address to the caller (frontend)
    jsonResponse(w, map[string]string{"address": address})
}
```

**Important**: Never return the private key to the frontend – only the public deposit address. All private key handling stays on the server.

## Deposit Monitoring (Polling for TRC-20 Transactions)

To detect user deposits, the backend will run a **periodic job** that polls the blockchain for new USDT transactions into our generated addresses. There is no native webhook in TronGrid’s free API (no push notifications), so we will implement polling at regular intervals.

### Polling Strategy and 6-Block Confirmations

- **Interval**: Decide on a polling frequency (e.g., every 30 seconds or 1 minute). Tron’s block time is ~3 seconds, so blocks are frequent. Even with a 1-minute interval, we won’t miss transactions and can enforce 6 confirmations (~18 seconds) easily.

- **TronGrid Query**: We have two approaches:
  1. **Address-by-address**: Use TronGrid’s account API for each user address to get TRC-20 transactions. This is straightforward but inefficient if many addresses (lots of API calls).
  2. **Contract events**: Query the USDT contract’s **Transfer** events and filter for addresses we own. TronGrid provides a **contract events endpoint** to fetch events by contract. We can use the USDT contract address and scan recent events:
     ```http
     GET https://api.trongrid.io/v1/contracts/TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t/events?only_confirmed=true&since=<LAST_TIMESTAMP>
     ```
     We can store a `last_timestamp` or `last_block` processed. Each poll, fetch events newer than that (the TronGrid API supports parameters like `min_timestamp` or `since` for event queries). Then filter the results in our code: for each event of type **Transfer**, check if the `to` address matches any in our `user_wallets` table.
     
     Using contract events in batches is efficient for many addresses, as we fetch all transfers in one call. **Note**: Ensure to handle pagination if many events; TronGrid’s `limit` parameter might need to be used if a single call can’t retrieve all new events.

- **6 Confirmations**: TronGrid’s `only_confirmed=true` filter will ensure we only get transactions that are already confirmed on-chain (Tron uses an asynchronous confirmation model). However, to be extra safe with reorgs, we wait for 6 block confirmations. In practice, Tron’s DPoS consensus makes reorgs rare, but we adhere to the requirement:
  - Each event comes with a block number. Track the latest confirmed block (you can get latest block via `https://api.trongrid.io/wallet/getnowblock`).
  - Only credit the deposit if `currentBlockNumber - event.blockNumber >= 6`. We may need to cache events until enough blocks have passed. Alternatively, delay processing events by one polling cycle or two, ensuring ~6 blocks have passed.
  
### Implementation: Monitoring Service

We can implement the polling in the backend as a **goroutine** that runs on application start (or a cron job). For example, in `main.go`, after initializing the server, start a background loop:

```go
go func() {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    var lastTimestamp int64 = 0  // track last processed timestamp (Unix ms)
    for range ticker.C {
        // Fetch events from TronGrid since lastTimestamp
        url := fmt.Sprintf("https://api.trongrid.io/v1/contracts/%s/events?only_confirmed=true&min_timestamp=%d", USDT_CONTRACT, lastTimestamp)
        resp, err := http.Get(url)
        if err != nil { log.Println("TronGrid fetch error:", err); continue }
        var data TronGridEventsResponse
        json.NewDecoder(resp.Body).Decode(&data)
        resp.Body.Close()
        // Process each event
        for _, event := range data.Data {
            if event.EventName == "Transfer" {
                toAddr := event.Result.To  // this might be hex; convert to base58
                amount := event.Result.Value  // amount of USDT (in smallest unit if applicable)
                // Only handle if toAddr is one of our user deposit addresses:
                userId := lookupUserByAddress(toAddr)
                if userId != 0 {
                    // Check confirmations if needed (TronGrid only_confirmed likely already final)
                    currentBlock := data.LatestBlock  // assume API provides, or fetch separately
                    if currentBlock - event.BlockNumber >= 6 {
                        handleConfirmedDeposit(userId, amount, event.TransactionId)
                    } else {
                        // optionally: queue for next round when confirmed
                        pendingDeposits[event.TransactionId] = event
                    }
                }
            }
            // Update lastTimestamp to the max seen to avoid reprocessing events
            if event.Timestamp > lastTimestamp {
                lastTimestamp = event.Timestamp
            }
        }
    }
}()
```

In the above pseudocode:
- `lookupUserByAddress(toAddr)` checks our `user_wallets` table (e.g., cached in memory or query DB) for the given address to find the associated user.
- `handleConfirmedDeposit` will update the database (user balance and payments log) for a confirmed deposit.
- We maintain `lastTimestamp` so that each poll only retrieves new events. TronGrid returns events sorted by time if `order_by=timestamp,asc` is used.
- We might use a map `pendingDeposits` to hold events that have <6 confirmations if needed (but with `only_confirmed`, TronGrid might already ensure basic confirmation).

**Batch vs Individual Calls**: The above uses one call for all events. If the event approach is not feasible (due to TronGrid limits on event query), an alternative is to batch user addresses into groups and call the account transactions API. For example, 10 addresses per batch, making parallel requests to `…/accounts/{address}/transactions/trc20?limit=...&only_confirmed=true&contract_address=TR7NHq...` for each. This is less efficient but more straightforward. Given TronGrid’s suggestion, calling the TRC-20 endpoint for addresses ensures we get all token transfers.

**Catching Up on Missed Events**: If the service restarts, ensure `lastTimestamp` persists (e.g., store in DB or a file). Alternatively, on startup, set `lastTimestamp` to now minus a buffer (like 1 minute) to catch any recent events.

## Deposit Processing: Crediting User Balance and Logging Payment

Once a deposit transaction is identified and deemed confirmed, we need to update the user’s account:

1. **Update User Balance**: Increase the user’s balance in `webapp_users.balance` by the deposited amount. This ensures the funds are reflected in their account for use within the platform.

2. **Log the Payment**: Record the deposit in the `payments` table for transaction history and audit. The entry should include:
   - `user_id` – link to the user.
   - `amount` – the USDT amount deposited (ensure to convert to the platform’s expected unit, e.g., if using float or smallest currency unit).
   - `status` – mark as `"paid by crypto"` (or similar status indicating a completed crypto deposit).
   - `datetime` – timestamp of the deposit confirmation.

3. **Idempotency**: Ensure a transaction (by txID) is only processed once. You might add a uniqueness constraint or check in `payments` (e.g., a new column for `tx_id`) before inserting a new record. If a `tx_id` exists, skip to avoid double credit.

### Database Update Code

In the backend, create a function to handle confirmed deposits (as referenced in the polling loop above):

```go
func handleConfirmedDeposit(userID int, amount float64, txID string) {
    // Use a transaction to update balance and insert payment atomically
    tx, _ := db.PostgresEngine.Begin()
    defer tx.Rollback()
    // 1. Update user balance
    _, err := tx.Exec(`UPDATE webapp_users SET balance = balance + $1 WHERE id = $2`, amount, userID)
    if err != nil { log.Println("Balance update error:", err); return }
    // 2. Insert payment record
    _, err = tx.Exec(`INSERT INTO payments (user_id, amount, status, datetime, tx_id) 
                      VALUES ($1, $2, 'paid by crypto', NOW(), $3)`, userID, amount, txID)
    if err != nil { 
        if isUniqueViolation(err) {
            log.Println("Duplicate tx, already processed:", txID)
            tx.Commit(); return
        }
        log.Println("Insert payment error:", err); return 
    }
    tx.Commit()
    log.Printf("Deposited %.2f USDT for user %d (tx %s)\n", amount, userID, txID)
}
```

*(Here we assume adding a `tx_id` column to `payments` for uniqueness; see schema updates below. If not adding `tx_id`, ensure uniqueness by other means.)*

This will credit the user’s balance and record the payment in one transaction, so we don’t end up crediting without logging or vice versa. The `status='paid by crypto'` differentiates it from other payment methods (e.g., 'paid' might be used for fiat or other flows).

**PostgreSQL Schema Update (Payments)**: If the `payments` table doesn’t have a field to store the transaction hash and payment method, consider updating it:
```sql
ALTER TABLE payments 
    ADD COLUMN tx_id VARCHAR(100),
    ADD COLUMN method VARCHAR(20);
```
- `tx_id` can store the blockchain transaction hash (to avoid duplicates and for reference).
- `method` can store 'TRC20' or 'crypto' etc., but since `status` already covers 'paid by crypto', this might be optional. Alternatively, use `status` field to include more detail (but keep it concise, e.g., `status='paid (crypto)'` or separate field).

Logging the transaction in the database ensures the frontend can display it in the user’s payment history.

## Frontend Integration (hvmnd-web-app)

On the frontend, we need to expose the deposit address to the user and reflect their updated balance and payment history.

### Displaying the Deposit Address

Provide a section in the user’s account page (e.g., “Deposit Funds”) that shows their **TRC-20 USDT deposit address** (the one generated and stored earlier). Steps:

- Add a new API endpoint in **hvmnd-api** like `GET /api/v1/user/deposit-address` (secured by auth token) to retrieve the current user’s Tron USDT address. This will query `user_wallets` for the user. If not found (address not generated yet), the backend could either generate on the fly or return an error/flag for the frontend to prompt generation.

- In **hvmnd-web-app** (assuming a React or similar application):
  - Call the above API when the user navigates to the deposit section.
  - If an address is returned, display it along with a QR code (optional) and instructions: e.g., “Send TRC-20 USDT to this address. Your balance will update after 6 confirmations (~2 minutes).”
  - If no address exists (and perhaps the backend chooses not to auto-generate), include a button “Generate Deposit Address” that calls a POST endpoint to create one (which uses the `CreateCryptoWallet` handler described earlier).

**Example React Component (simplified):**
```jsx
function DepositWidget() {
  const [depositAddress, setDepositAddress] = useState("");
  useEffect(() => {
    api.get("/api/v1/user/deposit-address").then(res => {
      setDepositAddress(res.data.address);
    });
  }, []);
  
  return (
    <div className="deposit-section">
      <h3>Your USDT (TRC-20) Deposit Address:</h3>
      {depositAddress ? (
        <div>
          <code>{depositAddress}</code>
          {/* If desired, show QR code using a library by passing depositAddress */}
        </div>
      ) : (
        <button onClick={generateAddress}>Generate Deposit Address</button>
      )}
      <p>Send USDT on the Tron network (TRC-20) to this address. It will be credited after 6 confirmations.</p>
    </div>
  );
}
```

After the user sends USDT to the address, our backend will detect it and update their balance. We should ensure the frontend reflects the new balance and shows the transaction in history.

### Showing Updated Balance and Payment History

- The **balance** field in `webapp_users` is already used in the platform. Ensure that wherever the balance is displayed (dashboard, profile, etc.), it will show the increased amount after deposit. This likely happens automatically on next fetch of user data or if using a session token that contains balance, etc. We might trigger a refetch of the user info after a deposit or use WebSocket to notify the frontend of balance changes (optional enhancement).

- **Payment History**: The `payments` table is likely exposed via an API (e.g., `GET /api/v1/payments` or similar) that the frontend can call to list transactions. For example, there might be a section "Payment History" or "Transactions" in the web app. We should integrate the crypto deposit entries into this list:
  - They will appear with `status: "paid by crypto"` and an `amount` in USDT.
  - You may want to display a label like "USDT Deposit" or similar. If a `method` field was added, use that to differentiate. If not, parse the status or assume any entry with `tx_id` not null is a crypto deposit.
  - Example entry displayed: **Date** – **Amount** – **Method** – **Status**. E.g., `2025-03-04 14:30 | 100 USDT | Crypto (TRON) | Completed`.

No frontend changes are needed for withdrawals since we are not implementing that now. It may be wise to hide or disable any “Withdraw” UI if present, or clearly mark it as unavailable.

## Security Measures

Security must be considered at every step:

- **Private Key Protection**: As discussed, encrypt private keys in the database. Never store them plaintext. Restrict access to the table containing keys – even among developers or through admin interfaces – as much as possible. The encryption passphrase (ENV variable) should be rotated periodically and stored securely (e.g., in a vault or environment config, not in code).

- **Backend Access Control**: The API endpoints for generating or fetching deposit addresses should ensure the user can only access their own address (use authentication middleware). Similarly, the deposit monitoring service should be internal – not exposed as an endpoint – to prevent external triggers.

- **Least Privilege**: Since we are only handling deposits (incoming funds), we do not need to use the private keys at all in this phase. That means we do *not* need to load them in memory except possibly for address derivation/validation. We definitely do not use them to sign anything. This reduces risk. In future, if withdrawals are implemented, those operations should be restricted and carefully secured (e.g., multi-sig or admin approval).

- **Validate Incoming Data**: When processing events from TronGrid, validate that the `to` address and `amount` are well-formed and correspond to our records. Do not blindly trust external input. Also, consider using TronGrid’s checksum on addresses to ensure no address formatting issues.

- **Denial of Service & Rate Limits**: Polling TronGrid frequently could hit rate limits. Use batch calls as described and consider catching TronGrid errors. If TronGrid fails or is slow, implement exponential backoff or lower polling frequency to avoid overwhelming either side. Also, limit how many past events you process at once (if the service was down for a while, process in chunks to avoid memory spikes).

- **Logging and Monitoring**: Keep logs of deposit detections and balance updates. This helps in auditing. However, avoid logging sensitive info (never log private keys or full raw responses that may contain them). Log just high-level events like “User X credited Y USDT from TX Z”.

- **PostgreSQL Security**: If using the new `user_wallets` table, apply proper permissions. For example, if the application uses an ORM or direct SQL, ensure that only the server can read the `encrypted_privkey` – not exposed via any SELECT in the API responses. Only use it internally if needed (e.g., for future withdrawals).

By following these measures, we minimize the risk of key compromise and ensure the system deals safely with external data.

## Multi-Chain Support Design (Extensible Framework)

To support additional token standards like ERC-20 (Ethereum) and BEP-20 (Binance Smart Chain) in the future, we will design the system in a modular way:

- **Abstract the Blockchain Interactions**: Create an interface or service layer for blockchain operations. For example, define an interface `CryptoProvider` with methods like `GenerateAddress(userID)`, `GetDepositsSince(lastCheckpoint)`, etc. Implement this interface for each chain:
  - `TronProvider` (for TRC-20 on Tron) – uses TronGrid.
  - `EthProvider` (for ERC-20 on Ethereum) – would use an Ethereum node/infura and listen for ERC-20 transfers to addresses.
  - `BscProvider` (for BEP-20 on BSC) – uses a BSC node API.

  Each provider knows its token contract address (USDT contract differs by chain) and how to fetch events. The polling service can then iterate through all active providers.

- **Unified Wallet Storage**: The `user_wallets` table already includes a `chain` field. We can store multiple addresses per user, one for Tron, one for Ethereum, etc., under the same user. For example:
  ```sql
  user_id | chain   | token | address           | encrypted_privkey 
  --------+---------+-------+-------------------+-------------------
       5  | TRON    | USDT  | T...              | (encrypted key)
       5  | ETH     | USDT  | 0x...             | (encrypted key)
  ```
  When you add ERC-20, you’d generate an Ethereum address for each user (or on-demand) and add a row with `chain='ETH'`. The monitoring service for Ethereum would look for transfers to those addresses on the Ethereum network (likely via etherscan API or running a light node or using web3 filters).

- **Scalable Architecture**: If supporting multiple chains, consider running the deposit monitoring for each chain in parallel. You might even separate them into microservices if traffic is high (one service listens to Tron, another to Ethereum). But initially, a single service querying sequentially each provider (Tron, then Eth, etc.) every cycle is fine.

- **Configuration**: Use configuration files or environment variables to store chain-specific constants, such as RPC URLs (TronGrid URL, Infura URL), contract addresses (USDT on Tron vs Ethereum vs BSC), and confirmation requirements (6 is common, but some chains might need more; e.g., Ethereum often uses 12 confirmations for safety). This allows tweaking per chain easily.

By designing with these principles, when it’s time to integrate ERC-20 or BEP-20:
- Add the new provider implementation.
- Generate addresses for existing users on the new chain (possibly via migration script or on-demand).
- Extend the front-end to display the new deposit address (if needed) and update the monitoring loop to include the new chain.

The rest of the flow (balance credit, logging) can remain largely unchanged, since it operates on the generic `payments` table and `webapp_users.balance`. Just make sure to indicate in the `payments` record which chain the payment came from. Using the `method` or `status` field for clarity (e.g., status = "paid by crypto (ETH)") will help differentiate if users deposit via different chains.

## PostgreSQL Schema Updates Summary

To implement the above features, apply the following schema changes:

- **Crypto Networks Table**:
    Stores information about supported blockchain networks and tokens:
    ```sql
    CREATE TABLE public.crypto_networks (
      id serial4 NOT NULL,
      name varchar(50) NOT NULL,
      token_symbol varchar(10) NOT NULL,
      contract_address varchar(255) DEFAULT NULL::character varying NULL,
      network_fee numeric(20, 10) DEFAULT 0 NULL,
      created_at timestamptz DEFAULT CURRENT_TIMESTAMP NULL,
      CONSTRAINT crypto_networks_pkey PRIMARY KEY (id)
    );
    ```

- **Crypto Deposit Addresses Table** (for deposit addresses & keys):
  Stores user deposit addresses for each supported network:
  ```sql
  CREATE TABLE public.crypto_deposit_addresses (
    id serial4 NOT NULL,
    user_id int4 NULL,
    network_id int4 NULL,
    address varchar(255) NOT NULL,
    is_used bool DEFAULT false NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP NULL,
    used_at timestamptz NULL,
    CONSTRAINT crypto_deposit_addresses_pkey PRIMARY KEY (id),
    CONSTRAINT crypto_deposit_addresses_network_id_fkey FOREIGN KEY (network_id) REFERENCES public.crypto_networks(id) ON DELETE CASCADE,
    CONSTRAINT crypto_deposit_addresses_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.webapp_users(id) ON DELETE CASCADE
  );
  ```

- **Crypto Payment Transactions Table**:
    Records all incoming crypto deposits and their confirmation status:
    ```sql
    CREATE TABLE public.crypto_payment_transactions (
      id serial4 NOT NULL,
      user_id int4 NULL,
      deposit_address_id int4 NULL,
      network_id int4 NULL,
      amount numeric(20, 10) NOT NULL,
      token_symbol varchar(10) NOT NULL,
      transaction_hash varchar(255) DEFAULT NULL::character varying NULL,
      status varchar(50) NOT NULL,
      confirmations int4 DEFAULT 0 NULL,
      created_at timestamptz DEFAULT CURRENT_TIMESTAMP NULL,
      confirmed_at timestamptz NULL,
      CONSTRAINT crypto_payment_transactions_pkey PRIMARY KEY (id),
      CONSTRAINT crypto_payment_transactions_deposit_address_id_fkey FOREIGN KEY (deposit_address_id) REFERENCES public.crypto_deposit_addresses(id),
      CONSTRAINT crypto_payment_transactions_network_id_fkey FOREIGN KEY (network_id) REFERENCES public.crypto_networks(id),
      CONSTRAINT crypto_payment_transactions_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.webapp_users(id) ON DELETE CASCADE
    );
    ```

## Backend API Implementation (hvmnd-api)

With the database and design in place, the backend implementation consists of:

### 1. **Wallet Creation Endpoint**

Implement an endpoint to create or retrieve a deposit address:
- **HTTP GET /api/v1/user/deposit-address** – returns the existing Tron USDT deposit address for the authenticated user, or 404/empty if none.
- **HTTP POST /api/v1/user/deposit-address** – (if using on-demand generation) triggers creation of a new Tron address for the user (using `TronWeb.createAccount()` or similar in the server side), stores it, and returns it.

**Go Handler Example** (`handlers/payments.go` or `handlers/webapp_users.go`):
```go
func GetDepositAddress(w http.ResponseWriter, r *http.Request) {
    userID := auth.MustGetUserID(r)  // assume this retrieves authenticated user ID
    var address string
    err := db.PostgresEngine.QueryRow(`SELECT address FROM user_wallets 
                                       WHERE user_id=$1 AND chain='TRON' AND token='USDT'`, 
                                       userID).Scan(&address)
    if err == sql.ErrNoRows {
        http.Error(w, "No deposit address", http.StatusNotFound)
        return
    } else if err != nil {
        http.Error(w, "Server error", http.StatusInternalServerError)
        return
    }
    json.NewEncoder(w).Encode(map[string]string{"address": address})
}
```

For the **creation** (if needed):
```go
func CreateDepositAddress(w http.ResponseWriter, r *http.Request) {
    userID := auth.MustGetUserID(r)
    // Check if already exists
    var count int
    _ = db.PostgresEngine.QueryRow(`SELECT COUNT(*) FROM user_wallets WHERE user_id=$1 AND chain='TRON' AND token='USDT'`, userID).Scan(&count)
    if count > 0 {
        http.Error(w, "Address already exists", http.StatusBadRequest)
        return
    }
    // Generate new Tron address
    tronAccount, err := tronWebCreateAccount()  // this could call a Node script or use an integrated library
    if err != nil {
        http.Error(w, "Failed to generate address", http.StatusInternalServerError)
        return
    }
    address := tronAccount.Base58
    privKey := tronAccount.PrivateKey
    encKey, err := encryptPrivateKey(privKey, []byte(os.Getenv("ENC_KEY")))
    if err != nil {
        http.Error(w, "Failed to secure key", http.StatusInternalServerError)
        return
    }
    _, err = db.PostgresEngine.Exec(`INSERT INTO user_wallets(user_id, chain, token, address, encrypted_privkey) 
                                     VALUES($1,$2,$3,$4,$5)`, userID, "TRON", "USDT", address, encKey)
    if err != nil {
        http.Error(w, "DB insert error", http.StatusInternalServerError)
        return
    }
    json.NewEncoder(w).Encode(map[string]string{"address": address})
}
```

*(In practice, you might integrate address generation in Go itself. The `tronWebCreateAccount()` here is a placeholder for that logic.)*

- Ensure to protect these endpoints with authentication (only logged-in users can get/generate their address).
- Possibly integrate address generation into user registration flow to eagerly create a wallet, but on-demand is more efficient.

### 2. **Background Job for Monitoring Deposits**

As described in the **Deposit Monitoring** section, incorporate a background goroutine or scheduled task to regularly check TronGrid for new deposits. This likely lives in the initialization of the server:
```go
func StartDepositWatcher() {
    // code similar to the earlier pseudocode for polling TronGrid
}
```
Call `StartDepositWatcher()` from `main.go` after database initialization. Keep it running as long as the server runs. If using a separate service (for scaling), ensure it shares database access.

The watcher should use the `handleConfirmedDeposit` logic to update balances and insert payments when deposits come in.

### 3. **Balance Updates & Payment Logging**

We already implemented `handleConfirmedDeposit` above. We should integrate that function or inline its logic in the watcher. After updating DB, this function could also send a real-time notification to the user (e.g., via WebSocket or push message) to inform them of the new deposit – this is optional but improves UX.

Double-check that the `webapp_users.balance` field is of a type that can handle USDT values (likely a numeric or float). USDT has 6 decimals on Tron, but often platforms treat it as an integer in cents. We might use `numeric(18,6)` or similar. If it’s a float, be mindful of precision issues – consider using an integer number of micro-USDT internally.

### 4. **No Withdrawal Implementation**

We omit any withdrawal endpoints or processing. To avoid confusion:
- Do not expose any endpoint that could use the stored private keys to send funds out.
- If the UI has a withdrawal button, disable it or have it show a “coming soon” message.
- Perhaps add a check in the backend to reject any withdrawal-related requests if they somehow get called.

By **not** handling withdrawals, the private keys remain unused on the server. Users’ deposited USDT will remain in their deposit addresses. (In the future, one might build a system to periodically sweep those to a central cold wallet for security, but that’s beyond current scope. For now, if funds are needed to be spent or moved, an admin would have to manually access the keys.)

## Testing with Tron Testnet (Nile)

Before deploying to mainnet, test the integration on Tron’s test network:

- **Tron Testnet Selection**: Tron has multiple testnets; **Nile** is commonly used (or Shasta). We will use Nile here. TronGrid has a URL for testnet: `https://api.nile.trongrid.io` (just replace the host with Nile). In TronWeb, you can set `fullHost: "https://api.nile.trongrid.io"` to connect to testnet.

- **USDT on Testnet**: There is a test USDT (or similar stablecoin) deployed on Nile. For example, Tron’s documentation mentions a USDT test token on Nile with contract `TXYZopYRdj2D9XRtbG411XZZ3kM5VkAeBf` ([Get 2000 test coins - TRON | NILE TESTNET](https://nileex.io/join/getJoinPage#:~:text=Get%202000%20test%20coins%20,obtained%20by%20entering%20account%20address)). We can use that for testing deposits. Alternatively, deploy a simple TRC-20 token contract on Nile to simulate USDT.

- **Faucets**: Get test TRX from a faucet (needed to pay gas for any transactions on Tron testnet). Also obtain test USDT tokens:
  - Nile faucet for USDT: Tron provides a site (nileex.io as referenced) where you can request 1000 test USDT to be sent to an address ([Get 2000 test coins - TRON | NILE TESTNET](https://nileex.io/join/getJoinPage#:~:text=Get%202000%20test%20coins%20,obtained%20by%20entering%20account%20address)).
  - Send some of these tokens to the deposit address generated for a test user.

- **Configure the App for Testnet**:
  - In **hvmnd-api**, set environment variables or constants for TronGrid URL to the testnet URL.
  - Set the USDT contract address to the testnet USDT’s address.
  - Perhaps reduce confirmation requirement to 1 or 2 in config if you want faster tests (not necessary, 6 is fine on Nile too).
  - Generate a few user deposit addresses on test (these will be different from mainnet addresses).
  
- **Run the deposit watcher** and observe logs. Use a test user’s deposit address, and transfer (from a TronLink wallet or Tron IDE) a certain amount of test USDT to it. Then monitor:
  - Does TronGrid event show up after the transfer? The watcher should capture it within the polling interval.
  - Ensure after 6 blocks (~18s) the watcher calls `handleConfirmedDeposit` and the database updates.
  - Verify the user’s `balance` in DB is incremented correctly.
  - Call the `GET /payments` (if exists) or directly query DB to see the new payment record with status 'paid by crypto'.
  - Also check the frontend: refresh the user’s account page and see the new balance. The payment history should list the deposit. The deposit address remains the same (persisted).
  
- **Edge Cases**: Test sending two deposits in quick succession to the same address – ensure both get logged. Test what happens if an address receives a token other than USDT (should be ignored by our logic since we filter by contract). Also, simulate a small reorg (hard on Tron testnet, but one could artificially lower confirmation count to 0 and see if double processing occurs, etc.). 

- **Error Handling**: Shut down TronGrid or disconnect internet to simulate TronGrid failure – ensure our app doesn’t crash and resumes gracefully.

Once tests pass on testnet:
- Remove or reset any test configuration.
- Switch TronGrid URL back to mainnet.
- Ensure the USDT contract address is the mainnet one (TR7NHq...).
- Keep confirmation at 6 for production.
- All systems go for mainnet deployment.

## Conclusion

Following this guide, we have:
- Set up **TronGrid** as a blockchain provider for TRC-20 USDT.
- Implemented secure **wallet generation** for users, encrypting private keys and storing deposit addresses in PostgreSQL.
- Created a **polling mechanism** to monitor the Tron blockchain for deposits, using 6-block confirmation for safety and batching calls for efficiency.
- Processed confirmed deposits by **updating user balances** and logging the transactions in the `payments` table with status `'paid by crypto'`.
- Skipped withdrawal functionality (reducing security exposure) as per requirements.
- Designed the solution to be **extensible** for multiple chains, ensuring future ERC-20/BEP-20 integration is straightforward.
- Incorporated necessary **security measures** (encryption, access control, idempotency) to protect user assets.
- Verified the solution with **extensive testing on Tron’s testnet** before going live.

By following the step-by-step implementation and referencing the code snippets provided, the development team can efficiently and securely integrate TRC-20 USDT payments into **hvmnd-api** and **hvmnd-web-app**, enhancing the platform with crypto payment capabilities. 

