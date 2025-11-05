// Rollback migration: Drop indexes for conversations and messages

// Drop conversations collection indexes
db.conversations.dropIndex("created_at_-1");
db.conversations.dropIndex("updated_at_-1");

// Drop messages collection indexes
db.messages.dropIndex("conversation_id_1_created_at_1");
db.messages.dropIndex("role_1");
db.messages.dropIndex("created_at_-1");

// Drop compound index
db.messages.dropIndex("conversation_id_1_role_1_created_at_1");

print("Indexes dropped successfully");
