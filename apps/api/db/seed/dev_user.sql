-- Local development only. Never run against a production database.
-- Idempotent: safe to execute repeatedly.

INSERT INTO users (id, email, display_name, onboarded_at)
VALUES ('11111111-1111-4111-8111-111111111111', 'dev@fisca.local', 'Dev User', now())
ON CONFLICT (id) DO NOTHING;

INSERT INTO accounts (user_id, name, type, is_default)
VALUES ('11111111-1111-4111-8111-111111111111', 'Cash', 'cash', true)
ON CONFLICT DO NOTHING;

INSERT INTO categories (user_id, name, kind, icon, color, is_default) VALUES
    ('11111111-1111-4111-8111-111111111111', 'Makanan & Minuman', 'expense', 'utensils', '#FB7185', true),
    ('11111111-1111-4111-8111-111111111111', 'Transportasi', 'expense', 'car', '#38BDF8', true),
    ('11111111-1111-4111-8111-111111111111', 'Belanja', 'expense', 'shopping-bag', '#FBBF24', true),
    ('11111111-1111-4111-8111-111111111111', 'Tagihan & Utilitas', 'expense', 'receipt', '#6366F1', true),
    ('11111111-1111-4111-8111-111111111111', 'Hiburan', 'expense', 'popcorn', '#A78BFA', true),
    ('11111111-1111-4111-8111-111111111111', 'Kesehatan', 'expense', 'heart-pulse', '#F43F5E', true),
    ('11111111-1111-4111-8111-111111111111', 'Gaji', 'income', 'banknote', '#34D399', true),
    ('11111111-1111-4111-8111-111111111111', 'Freelance', 'income', 'laptop', '#10B981', true),
    ('11111111-1111-4111-8111-111111111111', 'Pemasukan Lain', 'income', 'plus-circle', '#94A3B8', true)
ON CONFLICT DO NOTHING;
