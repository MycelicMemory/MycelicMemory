"""
Ultrathink REST API Client for LoCoMo Benchmark
Provides interface for memory ingestion and semantic retrieval
"""

import requests
import json
import time
from typing import List, Dict, Optional, Tuple
from dataclasses import dataclass


@dataclass
class Memory:
    """Memory item from ultrathink."""
    id: str
    content: str
    importance: int
    tags: List[str]
    domain: Optional[str] = None
    created_at: Optional[str] = None
    similarity_score: Optional[float] = None


@dataclass
class RetrievalResult:
    """Result from semantic retrieval."""
    memory: Memory
    relevance_score: float
    similarity_score: Optional[float] = None


class UltrathinkClient:
    """REST API client for ultrathink memory system."""

    def __init__(self, base_url: str = "http://localhost:3002/api/v1", timeout: int = 30):
        """
        Initialize ultrathink client.

        Args:
            base_url: Ultrathink REST API base URL
            timeout: Request timeout in seconds
        """
        self.base_url = base_url
        self.timeout = timeout
        self.session = requests.Session()
        self.headers = {"Content-Type": "application/json"}

    def health_check(self) -> bool:
        """
        Verify ultrathink server is running and healthy.

        Returns:
            True if server is healthy, False otherwise
        """
        try:
            response = self.session.get(
                f"{self.base_url}/health",
                headers=self.headers,
                timeout=self.timeout
            )
            data = response.json()
            return data.get("success", False) and data.get("data", {}).get("status") == "healthy"
        except Exception as e:
            print(f"❌ Health check failed: {e}")
            return False

    def ingest_conversation(
        self,
        messages: List[Dict],
        session_id: str,
        domain: str = "locomo-benchmark"
    ) -> Tuple[List[str], float]:
        """
        Store conversation messages as memories in ultrathink.

        Args:
            messages: List of messages with 'role' and 'content'
            session_id: Session ID for tracking
            domain: Knowledge domain

        Returns:
            Tuple of (memory_ids, total_time) - List of created memory IDs and time taken
        """
        memory_ids = []
        start_time = time.time()

        for i, msg in enumerate(messages):
            try:
                content = msg.get("content", "")
                role = msg.get("role", "unknown")

                if not content.strip():
                    continue

                # Prepare memory payload
                payload = {
                    "content": content,
                    "importance": 5,  # Default importance
                    "tags": [role, "conversation-turn", f"position-{i}", f"session-{session_id}"],
                    "domain": domain,
                    "source": f"locomo-{session_id}-turn-{i}"
                }

                # POST to ultrathink
                response = self.session.post(
                    f"{self.base_url}/memories",
                    headers=self.headers,
                    json=payload,
                    timeout=self.timeout
                )

                if response.status_code == 200:
                    data = response.json()
                    if data.get("success"):
                        memory_id = data.get("data", {}).get("id")
                        if memory_id:
                            memory_ids.append(memory_id)

            except Exception as e:
                print(f"⚠️  Failed to ingest message {i}: {e}")
                continue

        elapsed = time.time() - start_time
        return memory_ids, elapsed

    def retrieve_memories(
        self,
        query: str,
        top_k: int = 10,
        use_ai: bool = True,
        min_similarity: float = 0.3
    ) -> Tuple[List[RetrievalResult], float]:
        """
        Semantically retrieve relevant memories for a query.

        Args:
            query: Query string
            top_k: Number of top memories to retrieve
            use_ai: Use AI-powered semantic search
            min_similarity: Minimum similarity threshold

        Returns:
            Tuple of (retrieved_memories, retrieval_time)
        """
        start_time = time.time()

        try:
            payload = {
                "query": query,
                "limit": top_k,
                "use_ai": use_ai,
                "response_format": "concise",  # Token optimization
                "min_similarity": min_similarity
            }

            response = self.session.post(
                f"{self.base_url}/memories/search",
                headers=self.headers,
                json=payload,
                timeout=self.timeout
            )

            elapsed = time.time() - start_time

            if response.status_code != 200:
                print(f"❌ Retrieval failed: HTTP {response.status_code}: {response.text}")
                return [], elapsed

            data = response.json()

            # Parse results - API returns flat array in "data" field
            results = []
            for item in data.get("data", []):
                mem = Memory(
                    id=item.get("id", ""),
                    content=item.get("summary", item.get("content", "")),  # API uses "summary" field
                    importance=item.get("importance", 5),
                    tags=item.get("tags", []),
                    domain=item.get("domain"),
                    created_at=item.get("created_at"),
                    similarity_score=item.get("relevance_score")  # Use relevance_score as similarity
                )
                result = RetrievalResult(
                    memory=mem,
                    relevance_score=item.get("relevance_score", 0.0),
                    similarity_score=item.get("relevance_score")
                )
                results.append(result)

            return results, elapsed

        except Exception as e:
            print(f"❌ Retrieval error: {e}")
            return [], time.time() - start_time

    def clear_session(self, session_id: str) -> Tuple[int, float]:
        """
        Delete all memories for a session.

        Args:
            session_id: Session ID to clear

        Returns:
            Tuple of (num_deleted, deletion_time)
        """
        start_time = time.time()

        try:
            # Get all memories for this session via tag search
            payload = {
                "query": f"session-{session_id}",
                "limit": 1000
            }

            response = self.session.post(
                f"{self.base_url}/memories/search",
                headers=self.headers,
                json=payload,
                timeout=self.timeout
            )

            if response.status_code != 200:
                return 0, time.time() - start_time

            data = response.json()
            memories = data.get("data", [])
            num_deleted = 0

            # Delete each memory
            for item in memories:
                mem_id = item.get("memory", {}).get("id")
                if mem_id:
                    try:
                        del_response = self.session.delete(
                            f"{self.base_url}/memories/{mem_id}",
                            headers=self.headers,
                            timeout=self.timeout
                        )
                        if del_response.status_code == 200:
                            num_deleted += 1
                    except Exception as e:
                        print(f"⚠️  Failed to delete memory {mem_id}: {e}")

            elapsed = time.time() - start_time
            return num_deleted, elapsed

        except Exception as e:
            print(f"❌ Cleanup error: {e}")
            return 0, time.time() - start_time

    def format_retrieved_as_context(self, results: List[RetrievalResult]) -> str:
        """
        Format retrieved memories into compact context string.

        Args:
            results: List of RetrievalResult objects

        Returns:
            Formatted context string
        """
        parts = []
        for i, result in enumerate(results, 1):
            # Include memory content and relevance score
            score = result.similarity_score or result.relevance_score
            parts.append(f"[Memory {i}] {result.memory.content}")
            if score:
                parts.append(f"(Relevance: {score:.2f})")

        return "\n".join(parts)

    def get_stats(self) -> Dict:
        """Get ultrathink server statistics."""
        try:
            response = self.session.get(
                f"{self.base_url}/stats",
                headers=self.headers,
                timeout=self.timeout
            )
            if response.status_code == 200:
                return response.json().get("data", {})
            return {}
        except Exception as e:
            print(f"⚠️  Failed to get stats: {e}")
            return {}
