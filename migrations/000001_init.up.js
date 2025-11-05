// Initial migration: Create indexes for conversations and messages

// Conversations collection indexes
db.conversations.createIndex({ "_id": 1 });
db.conversations.createIndex({ "created_at": -1 });
db.conversations.createIndex({ "updated_at": -1 });

// Messages collection indexes
db.messages.createIndex({ "conversation_id": 1, "created_at": 1 });
db.messages.createIndex({ "role": 1 });
db.messages.createIndex({ "created_at": -1 });

// Compound index for efficient queries
db.messages.createIndex({ "conversation_id": 1, "role": 1, "created_at": 1 });

print("Indexes created successfully");
