import os
import json
import urllib.request

base_url = os.environ.get("OPENAI_BASE_URL", "http://localhost:8080/v1")
session_id = os.environ.get("X_TRACEREPLAY_SESSION", "standalone_agent")

print(f"[Agent] Starting autonomous workflow with proxy: {base_url} (Session: {session_id})")

payload = {
    "model": "gpt-4o",
    "messages": [
        {"role": "user", "content": "Analyze repository security policy"}
    ]
}

req = urllib.request.Request(
    f"{base_url}/chat/completions",
    data=json.dumps(payload).encode("utf-8"),
    headers={
        "Content-Type": "application/json",
        "X-TraceReplay-Session": session_id,
    },
)

try:
    with urllib.request.urlopen(req) as resp:
        body = json.loads(resp.read().decode("utf-8"))
        print("[Agent] Received completion from TraceReplay proxy:")
        print(f"Content: {body['choices'][0]['message']['content']}")
except Exception as e:
    print(f"[Agent] Error: {e}")
