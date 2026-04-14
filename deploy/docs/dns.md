# DNS — linktor.dev

All records point to the production VPS (`<VPS_IP>`). Replace `<VPS_IP>` with the actual IPv4 (and `<VPS_IPV6>` if you have one).

| Type   | Name                | Value             | TTL  | Purpose                                     |
| ------ | ------------------- | ----------------- | ---- | ------------------------------------------- |
| `A`    | `@` (linktor.dev)   | `<VPS_IP>`        | 300  | Landing page                                |
| `AAAA` | `@`                 | `<VPS_IPV6>`      | 300  | Landing (IPv6, optional)                    |
| `A`    | `www`               | `<VPS_IP>`        | 300  | Landing (redirected to apex by Traefik)     |
| `A`    | `app`               | `<VPS_IP>`        | 300  | Admin dashboard                             |
| `A`    | `api`               | `<VPS_IP>`        | 300  | Backend API                                 |
| `A`    | `traefik`           | `<VPS_IP>`        | 300  | Traefik dashboard (basic auth)              |
| `A`    | `s3`                | `<VPS_IP>`        | 300  | MinIO S3 endpoint                           |
| `A`    | `s3-console`        | `<VPS_IP>`        | 300  | MinIO console                               |

Mail / verification (add as needed):

| Type   | Name              | Value                                                   | Notes                          |
| ------ | ----------------- | ------------------------------------------------------- | ------------------------------ |
| `MX`   | `@`               | (your provider, e.g. `mailgun`/`fastmail`)              | Outbound transactional mail    |
| `TXT`  | `@`               | `v=spf1 include:<provider> ~all`                        | SPF                            |
| `TXT`  | `_dmarc`          | `v=DMARC1; p=quarantine; rua=mailto:dmarc@linktor.dev`  | DMARC                          |
| `TXT`  | `<selector>._domainkey` | (DKIM key from provider)                          | DKIM                           |
| `CAA`  | `@`               | `0 issue "letsencrypt.org"`                             | Pin Let's Encrypt as the only CA |

## Verifying

After updating the records, confirm propagation before starting the stack:

```bash
for sub in @ www app api traefik s3 s3-console; do
  host=$( [ "$sub" = "@" ] && echo linktor.dev || echo $sub.linktor.dev )
  echo -n "$host → "
  dig +short "$host" A | head -n1
done
```

Each host must resolve to `<VPS_IP>` before Traefik can complete the ACME challenge. The `CAA` record is optional but strongly recommended — it prevents any other CA from issuing certificates for `linktor.dev`.
