#!/usr/bin/env bash
set -euo pipefail

# Deliberately opt-in: each model probe can create a billable request.
if [[ "${MODEL_PROBE_CONFIRM:-}" != "YES" ]]; then
  echo "Refusing to send model probes. Set MODEL_PROBE_CONFIRM=YES only for an isolated QA key." >&2
  exit 2
fi

base_url="${MODEL_PROBE_BASE_URL:-}"
api_key="${MODEL_PROBE_API_KEY:-}"
if [[ -z "$base_url" || -z "$api_key" ]]; then
  echo "MODEL_PROBE_BASE_URL and MODEL_PROBE_API_KEY are required." >&2
  exit 2
fi

base_url="${base_url%/}"
models_csv="${MODEL_PROBE_MODELS:-gpt-6-sol,gpt-6-luna,claude-opus-5-5,grok-4.7}"
IFS=',' read -r -a models <<< "$models_csv"

if [[ "${MODEL_PROBE_DRY_RUN:-0}" == "1" ]]; then
  printf 'dry-run: %s\n' "${models[*]}"
  exit 0
fi

tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

printf 'model\thttp_status\ttime_total_s\tresponse\n'
for model in "${models[@]}"; do
  model="${model//[[:space:]]/}"
  [[ -n "$model" ]] || continue
  body_file="$tmp_dir/body"
  meta_file="$tmp_dir/meta"
  payload=$(printf '{"model":"%s","messages":[{"role":"user","content":"Reply with OK."}],"max_tokens":1,"stream":false}' "$model")

  curl --silent --show-error --request POST \
    --connect-timeout 10 --max-time 45 \
    --header "Authorization: Bearer $api_key" \
    --header 'Content-Type: application/json' \
    --data "$payload" \
    --output "$body_file" \
    --write-out '%{http_code}\t%{time_total}\n' \
    "$base_url/v1/chat/completions" > "$meta_file" || true

  IFS=$'\t' read -r http_status time_total < "$meta_file" || true
  http_status="${http_status:-000}"
  time_total="${time_total:-unknown}"
  if [[ "$http_status" =~ ^2 ]]; then
    result="PASS"
  else
    result="FAIL"
  fi
  printf '%s\t%s\t%s\t%s\n' "$model" "$http_status" "$time_total" "$result"
done
