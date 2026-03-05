# Fine-tuning Phi-4-mini with QLoRA on Apple Silicon using MLX

**You can fine-tune Microsoft's 3.8B-parameter Phi-4-mini-instruct model for AI-powered git commit messages entirely on an M3 Pro MacBook with 18GB of unified memory**, using Apple's MLX framework with QLoRA and a 4-bit quantized model. Peak memory during training should stay around **6–8 GB**, leaving comfortable headroom. This guide covers the complete pipeline: environment setup, data preparation from CommitBench, QLoRA training, inference testing, adapter fusion, and serving via an OpenAI-compatible API. It also addresses switching to StarCoder2-7B for a code-specialized alternative.

MLX (`mlx-lm` version **0.30.7** as of February 2026) has matured into a full-featured local LLM toolkit. QLoRA is automatically triggered when a quantized model is detected — no special flag required. The entire workflow, from install to serving, uses five commands: `mlx_lm.lora`, `mlx_lm.generate`, `mlx_lm.fuse`, and `mlx_lm.server`.

---

## Setting up the MLX-LM environment

The `mlx-lm` package lives in its own dedicated repository at `ml-explore/mlx-lm` (it was split out from `mlx-examples` in 2025). Installation is straightforward:

```bash
pip install mlx-lm
```

Alternatively, `conda install -c conda-forge mlx-lm` works. The key system requirements are **macOS 14.0+ (Sonoma)** and an **Apple Silicon chip** (M1/M2/M3/M4). macOS 15+ is recommended for memory wiring optimizations that help with larger models. Python **3.10 or later** is required by the underlying MLX framework; Python 3.12 is the sweet spot for compatibility. Avoid running Python under Rosetta — verify with:

```bash
python -c "import platform; print(platform.processor())"
# Must print "arm", not "i386"
```

Once installed, you get these CLI entry points: `mlx_lm.generate`, `mlx_lm.chat`, `mlx_lm.convert`, `mlx_lm.lora`, `mlx_lm.fuse`, `mlx_lm.server`, `mlx_lm.cache_prompt`, `mlx_lm.merge`, and `mlx_lm.manage`. No additional setup is needed for Apple Silicon GPU acceleration — MLX uses unified memory natively.

---

## Choosing the right quantized Phi-4-mini model

The `mlx-community` organization on Hugging Face hosts three pre-quantized variants of `microsoft/Phi-4-mini-instruct`, all converted with mlx-lm v0.21.5:

| Model ID | Quantization | Disk size | Recommended for |
|---|---|---|---|
| **`mlx-community/Phi-4-mini-instruct-4bit`** | 4-bit | **2.16 GB** | Training on 18 GB (best headroom) |
| `mlx-community/Phi-4-mini-instruct-6bit` | 6-bit | 3.12 GB | Balance of quality and memory |
| `mlx-community/Phi-4-mini-instruct-8bit` | 8-bit | 4.08 GB | Highest quality, inference-focused |

**For QLoRA fine-tuning on 18 GB, the 4-bit model is the safest choice.** The base model weights consume only ~2.2 GB, and training overhead (optimizer states, activations, LoRA parameters) adds roughly 4–6 GB, putting peak usage around **6–8 GB**. The 8-bit variant works for inference but leaves less room during training. All three models include the tokenizer with a built-in `chat_template` that handles Phi-4-mini's special tokens (`<|system|>`, `<|user|>`, `<|assistant|>`, `<|end|>`) automatically — you never need to insert these tokens manually when using the `messages` JSONL format.

You can verify the model loads correctly with a quick inference test:

```bash
mlx_lm.generate --model mlx-community/Phi-4-mini-instruct-4bit \
    --prompt "Write a git commit message for adding a login form"
```

---

## Preparing CommitBench data for training

The **Maxscha/commitbench** dataset on Hugging Face contains **~1.66 million** commit (diff, message) pairs across six programming languages (Java, Python, Go, JavaScript, PHP, Ruby). Each row has six columns: `hash`, `diff`, `message`, `project`, `split`, and `diff_languages`. The dataset is split into ~1.17M train, ~250K validation, and ~250K test examples.

### Using the built-in training pipeline (recommended)

The `prepare-training` command applies commit-massage's full diff preprocessing pipeline — the same noise filtering, file importance ranking, and smart truncation used at inference time — to produce training data in OpenAI chat completion JSONL format. This ensures **training-serving consistency**: the model trains on data that exactly matches what it sees in production.

First, export CommitBench splits to raw JSONL (each line needs `diff` and `message` fields; `patch`/`subject`/`commit_message` aliases also work):

```python
import json
from datasets import load_dataset

ds = load_dataset("Maxscha/commitbench")

for split, path in [("train", "raw_train.jsonl"), ("validation", "raw_valid.jsonl"), ("test", "raw_test.jsonl")]:
    with open(path, "w") as f:
        for row in ds[split]:
            f.write(json.dumps({"diff": row["diff"], "message": row["message"]}) + "\n")
```

Then run each split through the pipeline:

```bash
commit-massage prepare-training raw_train.jsonl data/train.jsonl
commit-massage prepare-training raw_valid.jsonl data/valid.jsonl
commit-massage prepare-training raw_test.jsonl data/test.jsonl
```

The pipeline:
- Filters noise (lock files, generated code, binary files, vendored code) and replaces them with placeholders
- Ranks files by importance (source > config > test > docs > style) and truncates lowest-priority files first when over the 20k character budget
- Derives `git diff --stat`-style file change summaries from the raw diff
- Builds the user message in the same format as the inference path: `Files changed:` + stats + `Diff:` + processed diff
- Wraps everything in OpenAI chat completion format with the production system prompt
- Skips entries where the diff is empty or contains only noise after filtering
- Reports statistics (total/written/skipped) to stderr

Each output line is a complete training example:

```json
{"messages":[{"role":"system","content":"..."},{"role":"user","content":"Files changed:\n main.go | 5 +++--\n...\n\nDiff:\ndiff --git a/main.go..."},{"role":"assistant","content":"feat: add validation"}]}
```

If you need to subsample for an initial fine-tune (recommended for 18 GB), truncate the raw input JSONL before running `prepare-training`, or use `head`:

```bash
head -10000 raw_train.jsonl | commit-massage prepare-training /dev/stdin data/train.jsonl
```

### Alternative: manual Python conversion

If you want full control over the data format or need to customize the system prompt, you can convert CommitBench directly with Python. Note that this bypasses commit-massage's diff preprocessing, so training data won't exactly match inference-time input:

```python
import json
from datasets import load_dataset

ds = load_dataset("Maxscha/commitbench")

SYSTEM_PROMPT = (
    "You are a helpful assistant that writes concise, descriptive git commit "
    "messages. Given a git diff, generate an appropriate commit message."
)

def to_chat_jsonl(split_data, output_path, max_diff_chars=4000, max_examples=10000):
    with open(output_path, "w") as f:
        count = 0
        for row in split_data:
            if len(row["diff"]) > max_diff_chars:
                continue  # Skip very long diffs to stay within max_seq_length
            example = {
                "messages": [
                    {"role": "system", "content": SYSTEM_PROMPT},
                    {"role": "user", "content": row["diff"]},
                    {"role": "assistant", "content": row["message"]}
                ]
            }
            f.write(json.dumps(example) + "\n")
            count += 1
            if count >= max_examples:
                break
    print(f"Wrote {count} examples to {output_path}")

to_chat_jsonl(ds["train"], "data/train.jsonl", max_examples=10000)
to_chat_jsonl(ds["validation"], "data/valid.jsonl", max_examples=1000)
to_chat_jsonl(ds["test"], "data/test.jsonl", max_examples=1000)
```

### Data format notes

Each JSONL line must be a single-line JSON object with a `messages` key. MLX auto-detects the format and applies the model's chat template during tokenization. Subsampling to 10K training examples is practical for an initial fine-tune on 18 GB — you can scale up once you validate the pipeline.

The resulting files go in a `data/` directory:
```
data/
├── train.jsonl
├── valid.jsonl
└── test.jsonl
```

---

## QLoRA training configuration and execution

The core training command is `mlx_lm.lora`. When it detects a quantized model (like the 4-bit Phi-4-mini), **QLoRA is triggered automatically** — no flag needed. LoRA-specific parameters like `rank`, `scale`, `dropout`, and target layer `keys` are configured exclusively through a **YAML config file** (they have no CLI equivalents).

Create `lora_config.yaml`:

```yaml
# Model - using pre-quantized 4-bit triggers QLoRA automatically
model: "mlx-community/Phi-4-mini-instruct-4bit"

# Training flags
train: true
test: false
data: "./data"
seed: 42

# Architecture
fine_tune_type: lora
num_layers: 16           # Number of model layers to apply LoRA to

# Training hyperparameters
batch_size: 1            # Keep at 1 for 18GB memory safety
grad_accumulation_steps: 4   # Effective batch size = 4
iters: 1000
learning_rate: 2e-4      # Higher LR works well with QLoRA
max_seq_length: 2048

# Memory optimization
grad_checkpoint: true    # Trades compute for memory

# Logging and saving
steps_per_report: 10
steps_per_eval: 100
val_batches: 25
save_every: 200
adapter_path: "adapters"

# LoRA-specific parameters (config file only)
lora_parameters:
  keys:
    - "self_attn.q_proj"
    - "self_attn.k_proj"
    - "self_attn.v_proj"
    - "self_attn.o_proj"
    - "mlp.gate_proj"
    - "mlp.up_proj"
    - "mlp.down_proj"
  rank: 8
  scale: 10.0
  dropout: 0.05

# Optional: learning rate schedule
# lr_schedule:
#   name: cosine_decay
#   warmup: 100
#   warmup_init: 1e-7
#   arguments: [1e-5, 1000, 1e-7]
```

Launch training with a single command:

```bash
mlx_lm.lora --config lora_config.yaml
```

Any CLI argument overrides the corresponding YAML value, so you can experiment quickly:

```bash
mlx_lm.lora --config lora_config.yaml --iters 500 --learning-rate 1e-4
```

**Hyperparameter rationale for Phi-4-mini on 18 GB**: `batch_size: 1` with `grad_accumulation_steps: 4` gives an effective batch size of 4 without the memory cost. A learning rate of **2e-4** is common for QLoRA (higher than full fine-tuning because LoRA adapters have fewer parameters to update). `rank: 8` is the standard starting point for a 3.8B model — sufficient for task-specific adaptation like commit message generation. Targeting all seven projection layers (`q, k, v, o, gate, up, down`) gives broad coverage. The `--grad-checkpoint` flag enables gradient checkpointing, which recomputes activations during the backward pass instead of storing them, **reducing peak memory by roughly 30–40%** at the cost of ~20% slower training.

Training will output loss metrics every 10 steps and run validation every 100 steps. Adapters are saved as `adapters.safetensors` in the `adapters/` directory. For a 10K-example dataset with 1,000 iterations, expect training to take **30–90 minutes** on an M3 Pro depending on sequence lengths. You can also log to Weights & Biases by adding `--report-to wandb`.

---

## Testing inference with the trained adapter

After training completes, test the adapter without fusing it:

```bash
mlx_lm.generate \
    --model mlx-community/Phi-4-mini-instruct-4bit \
    --adapter-path ./adapters \
    --max-tokens 200 \
    --prompt "<|system|>You are a helpful assistant that writes git commit messages.<|end|>
<|user|>diff --git a/src/auth.py b/src/auth.py
--- a/src/auth.py
+++ b/src/auth.py
@@ -15,6 +15,12 @@ class AuthService:
+    def validate_token(self, token: str) -> bool:
+        if not token:
+            return False
+        return self.jwt_service.verify(token)
<|end|>
<|assistant|>"
```

For a cleaner workflow, use `mlx_lm.chat` for interactive testing:

```bash
mlx_lm.chat \
    --model mlx-community/Phi-4-mini-instruct-4bit \
    --adapter-path ./adapters
```

You can also evaluate on the held-out test set to get perplexity metrics:

```bash
mlx_lm.lora \
    --model mlx-community/Phi-4-mini-instruct-4bit \
    --adapter-path ./adapters \
    --data ./data \
    --test
```

---

## Fusing adapters and serving the model

Once you're satisfied with the fine-tuned model, fuse the LoRA adapters into the base weights to create a standalone model:

```bash
mlx_lm.fuse \
    --model mlx-community/Phi-4-mini-instruct-4bit \
    --adapter-path ./adapters \
    --save-path ./fused_model
```

This produces a complete model in `fused_model/` that can be loaded without specifying adapter paths. If you want to convert to full precision (e.g., for GGUF export), add `--de-quantize`. GGUF export is currently limited to Llama/Mistral/Mixtral architectures, so for Phi-4-mini you'd stay in MLX safetensors format or use a separate conversion pipeline.

**Serving with an OpenAI-compatible API** takes one command:

```bash
mlx_lm.server \
    --model ./fused_model \
    --host 127.0.0.1 \
    --port 8080
```

This starts an HTTP server implementing the OpenAI chat completions API. Call it from any application:

```bash
curl http://localhost:8080/v1/chat/completions \
    -H "Content-Type: application/json" \
    -d '{
        "messages": [
            {"role": "system", "content": "Generate a concise git commit message for this diff."},
            {"role": "user", "content": "diff --git a/README.md b/README.md\n+Added installation instructions"}
        ],
        "temperature": 0.3,
        "max_completion_tokens": 100
    }'
```

You can also serve the base model with a live adapter (without fusing):

```bash
mlx_lm.server \
    --model mlx-community/Phi-4-mini-instruct-4bit \
    --adapter-path ./adapters \
    --port 8080
```

For a git hook or CLI integration, pipe `git diff --cached` into a curl call to this server. The `temperature` of **0.3** keeps commit messages deterministic and focused. The server is single-threaded and not production-hardened, but it works well for local development tooling.

---

## Switching to StarCoder2-7B changes the data format entirely

StarCoder2-7B (`mlx-community/starcoder2-7b-4bit`, **4.36 GB** at 4-bit) is a **code completion model**, not a chat model. This fundamentally changes the data pipeline:

**Data format**: StarCoder2 uses the **`text` field** (or `prompt`/`completion` pair) instead of `messages`. Convert CommitBench to completion-style JSONL:

```python
# For StarCoder2: completion format
example = {
    "text": f"<commit_diff>\n{row['diff']}\n<commit_message>\n{row['message']}"
}
# OR prompt/completion format:
example = {
    "prompt": f"<commit_diff>\n{row['diff']}\n<commit_message>\n",
    "completion": row["message"]
}
```

**Memory**: At 4.36 GB base weight, StarCoder2-7B QLoRA training on 18 GB is tight. You'll need `batch_size: 1`, `grad_checkpoint: true`, `max_seq_length: 1024`, and possibly `num_layers: 8` instead of 16. Peak memory will likely be **10–14 GB**. **Hyperparameters**: A 7B model benefits from a lower learning rate (**1e-5**) and potentially higher rank (16) compared to the 3.8B Phi-4-mini. StarCoder2 has a 16K context window with sliding window attention of 4K, so keep sequences well under 4K for training. **Key trade-off**: StarCoder2 understands code structure better (trained on 3.5T tokens of code), but Phi-4-mini-instruct follows instructions more naturally and produces more human-readable commit messages out of the box.

---

## Memory management and troubleshooting on 18 GB

Running QLoRA on an 18 GB M3 Pro is viable but requires attention. Here's a priority-ordered list of memory reduction levers:

- **`batch_size: 1`** is the single most impactful setting — each additional batch element roughly doubles activation memory. Use `grad_accumulation_steps` to compensate for effective batch size.
- **`grad_checkpoint: true`** cuts activation memory by ~30–40% at the cost of ~20% longer training.
- **`max_seq_length: 1024`** or even 512 dramatically reduces memory for long inputs like diffs. Filter your dataset to exclude examples that would exceed this limit.
- **`num_layers: 8`** instead of 16 reduces the number of LoRA adapter matrices, though it may hurt quality.
- **Lower LoRA rank** (4 instead of 8) reduces adapter parameter count.

**Monitoring memory** during training: use `Activity Monitor > Memory` or run `sudo memory_pressure` in a terminal. From Python, you can check programmatically:

```python
import mlx.core as mx
info = mx.metal.get_memory_info()
print(f"Active: {info['active_memory'] / 1e9:.2f} GB")
print(f"Peak: {info['peak_memory'] / 1e9:.2f} GB")
```

**If training OOMs**: MLX won't crash with a clear error — instead, macOS will start heavy swap usage and the system becomes unresponsive. Watch for sudden slowdowns. If this happens, kill the process and reduce `max_seq_length` or `num_layers` first. On macOS 15+, you can wire GPU memory with `sudo sysctl iogpu.wired_limit_mb=12000` to prevent the OS from reclaiming memory during training.

**If training is too slow**: the most common cause is excessive sequence lengths causing large matrix operations. Shorter sequences train faster. Also ensure no other GPU-intensive applications are running. On the M3 Pro, expect roughly **100–250 tokens/second** during training and **20–40 tokens/second** during generation.

**Known issues**: some models (particularly Llama 3.x variants) have reported memory leaks during training where memory grows progressively across iterations. This has not been widely reported for Phi-4 models, but if you observe it, save checkpoints frequently with `save_every: 100` and resume from the last checkpoint with `--resume-adapter-file adapters/adapters.safetensors`.

---

## Conclusion

The full pipeline for fine-tuning Phi-4-mini on an M3 Pro MacBook with 18 GB boils down to five commands after data preparation: `mlx_lm.lora` (train), `mlx_lm.generate` (test), `mlx_lm.lora --test` (evaluate), `mlx_lm.fuse` (merge), and `mlx_lm.server` (deploy). The most critical configuration choices are using the **4-bit quantized model** (`mlx-community/Phi-4-mini-instruct-4bit`) to trigger QLoRA automatically, keeping **`batch_size: 1`** with gradient accumulation, and enabling **`grad_checkpoint: true`** — together these keep peak memory well under 10 GB.

For commit message generation specifically, the instruction-tuned Phi-4-mini is a stronger starting point than StarCoder2 because it naturally follows the "given a diff, write a message" instruction format. Start with 1,000 iterations on 10K filtered CommitBench examples, evaluate quality manually, then scale data and iterations based on results. The `--mask-prompt` flag (which computes loss only on the assistant's response, not the diff input) is worth experimenting with to focus learning on message generation rather than diff reconstruction. Once fused and served locally, the model integrates into any git workflow via a simple curl call to the OpenAI-compatible API on `localhost:8080`.
