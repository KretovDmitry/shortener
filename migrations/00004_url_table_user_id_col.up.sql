ALTER TABLE IF EXISTS public.url
    ADD COLUMN IF NOT EXISTS user_id UUID DEFAULT gen_random_uuid ();

