-- Transaction Service Database: finance_transaction
-- Таблица категорий
CREATE TABLE IF NOT EXISTS categories (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,
    name VARCHAR(100) NOT NULL,
    type VARCHAR(20) NOT NULL CHECK (type IN ('income', 'expense')),
    color VARCHAR(20),
    icon VARCHAR(50),
    parent_id VARCHAR(36) REFERENCES categories(id) ON DELETE SET NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_categories_user_id ON categories(user_id);
CREATE INDEX IF NOT EXISTS idx_categories_type ON categories(type);

-- Таблица транзакций
CREATE TABLE IF NOT EXISTS transactions (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,
    amount DECIMAL(15,2) NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'RUB',
    type VARCHAR(20) NOT NULL CHECK (type IN ('income', 'expense')),
    category_id VARCHAR(36) REFERENCES categories(id) ON DELETE SET NULL,
    description TEXT,
    date TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    tags TEXT[],
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_transactions_user_id ON transactions(user_id);
CREATE INDEX IF NOT EXISTS idx_transactions_date ON transactions(date);
CREATE INDEX IF NOT EXISTS idx_transactions_type ON transactions(type);
CREATE INDEX IF NOT EXISTS idx_transactions_category_id ON transactions(category_id);
CREATE INDEX IF NOT EXISTS idx_transactions_tags ON transactions USING GIN(tags);

-- Дефолтные категории
INSERT INTO categories (id, user_id, name, type, color, icon) VALUES
    ('default-salary', 'system', 'Зарплата', 'income', '#4CAF50', '💰'),
    ('default-business', 'system', 'Бизнес', 'income', '#8BC34A', '💼'),
    ('default-food', 'system', 'Еда', 'expense', '#FF5722', '🍔'),
    ('default-transport', 'system', 'Транспорт', 'expense', '#2196F3', '🚗'),
    ('default-entertainment', 'system', 'Развлечения', 'expense', '#9C27B0', '🎮'),
    ('default-health', 'system', 'Здоровье', 'expense', '#F44336', '💊'),
    ('default-education', 'system', 'Образование', 'expense', '#3F51B5', '📚'),
    ('default-other', 'system', 'Прочее', 'expense', '#9E9E9E', '📦')
ON CONFLICT (id) DO NOTHING;
