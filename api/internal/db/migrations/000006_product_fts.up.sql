-- Full-text search for storefront catalog search (FR-6.3, MH-402).
ALTER TABLE products
    ADD COLUMN search_tsv tsvector
    GENERATED ALWAYS AS (
        to_tsvector('simple',
            coalesce(product_name,'') || ' ' ||
            coalesce(description,'') || ' ' ||
            coalesce(sku,''))
    ) STORED;
CREATE INDEX idx_products_search_tsv ON products USING GIN (search_tsv);
