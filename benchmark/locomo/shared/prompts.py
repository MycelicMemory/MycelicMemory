"""
Prompts for LoCoMo benchmark evaluation.
Adapted from mem0ai/mem0 evaluation code.
"""

# Prompt for answering questions with graph-enhanced memories
ANSWER_PROMPT_GRAPH = """You are a memory assistant that helps answer questions based on stored memories.

You have access to memories from two speakers in a conversation, along with knowledge graph relations.

Speaker 1 ({speaker_1_user_id}) Memories:
{speaker_1_memories}

Speaker 2 ({speaker_2_user_id}) Memories:
{speaker_2_memories}

Knowledge Graph Relations:
{graph_relations}

Question: {question}

Instructions:
1. Analyze all provided memories from both speakers and use graph connections to understand user context.
2. Convert relative time references to specific dates, months, or years (e.g., "last year" -> actual year based on memory timestamps).
3. If there are contradictions, prioritize the most recent information.
4. Keep your answer under 5-6 words.
5. Do not confuse character names in memories with actual users.
6. Base your answer on explicit evidence in the memories, not assumptions.

Answer:"""

# Prompt for answering questions with memories (no graph)
ANSWER_PROMPT = """You are a memory assistant that helps answer questions based on stored memories.

You have access to memories from two speakers in a conversation.

Speaker 1 ({speaker_1_user_id}) Memories:
{speaker_1_memories}

Speaker 2 ({speaker_2_user_id}) Memories:
{speaker_2_memories}

Question: {question}

Instructions:
1. Examine all memories that contain information related to the question.
2. Convert relative time references to specific dates, months, or years (e.g., "last year" -> actual year based on memory timestamps).
3. If there are contradictions, prioritize the most recent information.
4. Keep your answer under 5-6 words.
5. Do not confuse character names in memories with actual users.
6. Base your answer on explicit evidence in the memories, not assumptions.

Answer:"""

# Prompt for multiple-choice questions with conversation context
ANSWER_PROMPT_ULTRATHINK = """You are a memory assistant that helps answer multiple-choice questions based on conversation history.

Conversation History:
{memories}

Question: {question}

Answer Options:
{choices}

Instructions:
1. Examine the conversation history carefully.
2. Identify which of the 10 answer options best answers the question.
3. Consider all context from the conversation history.
4. Convert relative time references to specific dates, months, or years if needed.
5. Return ONLY the choice index (0-9) that corresponds to the best answer.
6. Do not explain your choice, just return the index number.

Answer Index:"""

# Prompt for multiple-choice questions with session summaries
ANSWER_PROMPT_SUMMARY = """You are a memory assistant that helps answer multiple-choice questions based on session summaries.

Session Summaries:
{summaries}

Question: {question}

Answer Options:
{choices}

Instructions:
1. Analyze the session summaries to find information related to the question.
2. Identify which of the 10 answer options best answers the question.
3. Convert relative time references to specific dates, months, or years if needed.
4. If there are contradictions, prioritize the most recent information.
5. Return ONLY the choice index (0-9) that corresponds to the best answer.
6. Do not explain your choice, just return the index number.

Answer Index:"""
