import json
import os
from typing import Any, Dict
from pydantic import BaseModel


class ConfigLoader:
    def __init__(self, config_path: str = None):
        if not config_path:
            # Default to the shared config location
            base_dir = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
            config_path = os.path.join(base_dir, "configFiles", "dev.json")
        
        self.config_path = config_path
        self._config_data = self._load_config()

    def _load_config(self) -> Dict[str, Any]:
        if not os.path.exists(self.config_path):
            raise FileNotFoundError(f"Configuration file not found at: {self.config_path}")
        
        with open(self.config_path, "r") as f:
            return json.load(f)

    def get(self, key: str, default: Any = None) -> Any:
        return self._config_data.get(key, default)

    def get_all(self) -> Dict[str, Any]:
        return self._config_data

# Singleton instance for easy access
config = ConfigLoader()
