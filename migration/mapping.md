# Legacy → Target Field Mapping (MH-406)

## users → tenants
| Legacy (`users`) | Target (`tenants`) | Transform |
|---|---|---|
| `id` | `legacy_user_id` + new `id` (UUID) | mint UUID, keep legacy id for reconciliation |
| `username` | `username` | lower-trim |
| `email` | `email` | lower-trim |
| `password` | `password_hash` | **re-hash on first login**; store legacy hash flagged (Argon2id migration) |
| `first_name`/`last_name`/company | `full_name`/`store_name` | concat |

## mudahurus_products_category → categories
| `id`,`user_id`,`category`,`description` | `id`(UUID),`tenant_id`,`name`,`description` |

## mudahurus_products → products
| `sku`,`product_name`,`description`,`unit_price`,`category_id`,`status`,`image` | same; `image`→`image_key` after storage backfill; `url_slug` derived |

## mudahurus_orders → orders + order_items + payments (NORMALIZE)
| Legacy flat columns | Target |
|---|---|
| `sku,quantity,unit_price` | one `order_items` row |
| `total_price,status,full_name,email,contact_no,additional_notes,expired_date,insert_date` | `orders` header (shipping fields → `shipping_address` JSONB) |
| `mailing_addr,mailing_addr2,city,postcode,state` | `orders.shipping_address` JSONB |
| `payment_image_proof` | one `payments` row (`proof_key`, status derived from order status) |

## mudahurus_customers → customers
Direct column copy; `ic_no/dob/contact_no` flagged PII.

## mudahurus_coupons → coupons
| `campaign,description,product_id,expired_date` | same |

## general_logs → access_logs
Best-effort copy of ip/referrer/url/uri/time (parameterized insert; the legacy
string-interpolated writer is **not** ported).
