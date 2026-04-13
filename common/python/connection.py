from motor.motor_asyncio import AsyncIOMotorClient
import logging

logger = logging.getLogger(__name__)

class MongoDBConnection:
    def __init__(self):
        self._client: AsyncIOMotorClient | None = None

    async def connect(self, uri: str) -> None:
        """Open the MongoDB connection."""
        try:
            self._client = AsyncIOMotorClient(uri)
            # Check the connection
            await self._client.admin.command("ping")
            logger.info("Connected to MongoDB successfully")
        except Exception as e:
            logger.error(f"Failed to connect to MongoDB: {e}")
            raise e

    def close(self) -> None:
        """Close the MongoDB connection."""
        if self._client:
            self._client.close()
            self._client = None
            logger.info("MongoDB connection closed")

    def get_client(self) -> AsyncIOMotorClient:
        if self._client is None:
            raise RuntimeError("Database not initialised. Call connect() first.")
        return self._client

# Singleton instance
db_connection = MongoDBConnection()
