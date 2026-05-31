ALTER TABLE billing_request_holds
    ADD COLUMN IF NOT EXISTS conversion_breakdown JSONB NOT NULL DEFAULT '{}'::jsonb;

ALTER TABLE billing_request_holds
    ADD COLUMN IF NOT EXISTS conversion_policy JSONB NOT NULL DEFAULT '{}'::jsonb;

COMMENT ON COLUMN billing_request_holds.conversion_breakdown IS 'Wallet currencies debited by request hold/reserve, used to settle or release converted balances safely';

COMMENT ON COLUMN billing_request_holds.conversion_policy IS 'Currency conversion settings captured when the request hold was reserved';
