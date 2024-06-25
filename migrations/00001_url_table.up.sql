CREATE TABLE IF NOT EXISTS public.url (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid (),
    short_url varchar(255) NOT NULL,
    original_url text NOT NULL
);

