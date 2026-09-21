#!/usr/bin/env python3
"""Explicitly approved, minimal live smoke calls. Never prints or saves the key.

Run on the production host as an operator with PostgreSQL read access. A failed
or ambiguous call is NOT retried automatically. Use only an existing owner/admin
test key, not a customer key. No model alias substitution is performed here.
"""
import argparse
import datetime
import json
import subprocess
import urllib.error
import urllib.request


def query(sql):
    return subprocess.run(
        ["sudo", "-u", "postgres", "psql", "-X", "-d", "sub2api", "-At",
         "-v", "ON_ERROR_STOP=1", "-c", sql],
        check=True, capture_output=True, text=True, cwd="/",
    ).stdout.strip()


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--key-id", required=True, type=int)
    parser.add_argument("--models", nargs="+", choices=["gpt-6", "gpt-6-astra", "gpt-5.1"], required=True)
    parser.add_argument("--execute-approved-paid-smoke", action="store_true")
    args = parser.parse_args()
    if not args.execute_approved_paid_smoke or args.key_id <= 0:
        raise SystemExit("Explicit paid-smoke flag and positive key ID required")
    key = query(f"""SELECT k.key FROM api_keys k JOIN users u ON u.id=k.user_id
        JOIN groups g ON g.id=k.group_id WHERE k.id={args.key_id}
        AND u.role='admin' AND u.deleted_at IS NULL AND u.status='active'
        AND k.status='active' AND k.deleted_at IS NULL
        AND (k.expires_at IS NULL OR k.expires_at>NOW())
        AND g.status='active' AND g.deleted_at IS NULL AND g.platform='openai';""")
    if not key or "\n" in key:
        raise SystemExit("Key is not an eligible admin-owned OpenAI test key")
    failed = False
    for model in dict.fromkeys(args.models):
        body = json.dumps({
            "model": model, "input": [{"role": "user", "content": [
                {"type": "input_text", "text": "Reply with exactly OK."}]}],
            "max_output_tokens": 128, "reasoning": {"effort": "low"},
            "stream": False, "store": False,
        }).encode()
        request = urllib.request.Request("https://api.ssxzapi.com/v1/responses", data=body, headers={
            "Authorization": "Bearer " + key, "Content-Type": "application/json",
            "User-Agent": "SSXZ-approved-release-smoke/0.2.7",
        })
        result = {"requested_model": model, "started_utc": datetime.datetime.now(datetime.timezone.utc).isoformat()}
        try:
            with urllib.request.urlopen(request, timeout=120) as response:
                result["http_status"] = response.status
                data = json.loads(response.read(2 * 1024 * 1024))
            texts = [content.get("text", "") for output in data.get("output", [])
                     for content in output.get("content", []) if content.get("type") == "output_text"]
            result.update({"response_status": data.get("status"), "returned_model": data.get("model"),
                           "exact_ok_text": "".join(texts).strip() == "OK", "usage": data.get("usage"),
                           "response_id": data.get("id")})
            returned = data.get("model") or ""
            result["model_family_matches"] = returned == model or returned.startswith(model + "-")
            result["passed"] = (result["http_status"] == 200 and data.get("status") == "completed"
                                and result["exact_ok_text"] and result["model_family_matches"]
                                and not data.get("error"))
        except urllib.error.HTTPError as error:
            result["http_status"] = error.code
            try:
                data = json.loads(error.read(1024 * 1024))
                detail = data.get("error", {})
                result["error_code"] = detail.get("code") if isinstance(detail, dict) else None
                result["error_type"] = detail.get("type") if isinstance(detail, dict) else None
            except (ValueError, AttributeError):
                result["error_code"] = "non_json_error"
            result["passed"] = False
        except (urllib.error.URLError, TimeoutError, ValueError, OSError) as error:
            result.update({"passed": False, "error_type": type(error).__name__, "outcome_unknown": True})
        print(json.dumps(result, ensure_ascii=False), flush=True)
        failed |= not result["passed"]
    raise SystemExit(1 if failed else 0)


if __name__ == "__main__":
    main()
