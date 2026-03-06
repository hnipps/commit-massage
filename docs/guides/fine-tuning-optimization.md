# Fine-tuning optimization guide

This guide walks through the iterative process of fine-tuning a small language model (Phi-4-mini via QLoRA on Apple Silicon) to generate conventional commit messages. It assumes you've read the [MLX fine-tuning setup guide](mlx-fine-tuning.md) and have the environment working. The focus here is on **how to diagnose problems, iterate, and systematically improve results**.

---

## The fine-tuning feedback loop

Fine-tuning is not a one-shot process. It's an iterative loop:

```
1. Prepare data  →  2. Train  →  3. Evaluate  →  4. Diagnose  →  (repeat)
```

Each cycle, you change **one thing** — the data, a hyperparameter, or the prompt — and measure whether it helped. Changing multiple things at once makes it impossible to know what worked.

---

## Step 1: Get your data right (this is 80% of the work)

The single most impactful thing you can do is improve your training data. A model will faithfully reproduce whatever patterns are in the data — garbage in, garbage out.

### Use the filtered pipeline, not raw data

The `prepare-training` command applies `ValidateMessage` which only keeps commit messages that already follow conventional commit format (`type(scope): description`). Raw CommitBench data is full of messages like `"fixed bug"`, `"update readme"`, `"refactoring to use X"` — none of these follow the format. If you train on them, the model learns the wrong style.

```bash
# Re-generate training data with the full pipeline (filters + diff preprocessing)
head -50000 ml/raw_train.jsonl | commit-massage prepare-training /dev/stdin ml/data/train.jsonl
head -10000 ml/raw_valid.jsonl | commit-massage prepare-training /dev/stdin ml/data/valid.jsonl
head -10000 ml/raw_test.jsonl | commit-massage prepare-training /dev/stdin ml/data/test.jsonl
```

Check the skip statistics — the pipeline will report how many examples were filtered out and why. A high skip rate is normal and expected; it means the filter is working.

### Verify your data manually

Before training, always eyeball 10-20 random examples:

```bash
shuf ml/data/train.jsonl | head -10 | python3 -c "
import sys, json
for line in sys.stdin:
    msgs = json.loads(line)['messages']
    print('USER:', msgs[1]['content'][:80])
    print('ASST:', msgs[2]['content'])
    print('---')
"
```

Ask yourself:
- Does every assistant response start with a valid type prefix (`feat:`, `fix:`, etc.)?
- Are the messages lowercase?
- Are they concise (under 72 chars)?
- Does the diff in the user message look like what the model will see at inference time?

If the answers aren't all "yes", fix the data before training again.

### Data quantity guidelines

- **1K-5K examples**: Enough for a first experiment, trains in minutes. Good for validating the pipeline.
- **10K-30K examples**: The sweet spot for this task. Enough diversity to generalize without overfitting.
- **50K+ examples**: Diminishing returns for a narrow task like commit messages. Only worth it once everything else is optimized.

Start small (5K), get the format right, then scale up.

---

## Step 2: Train with sensible defaults

The `lora_config.yaml` in `ml/` is already a good starting point. Key settings:

```yaml
model: "mlx-community/Phi-4-mini-instruct-4bit"
batch_size: 1
grad_accumulation_steps: 4
iters: 1000
learning_rate: 2e-4
max_seq_length: 2048
grad_checkpoint: true
lora_parameters:
  rank: 8
  scale: 10.0
  dropout: 0.05
```

Run training:

```bash
cd ml
mlx_lm.lora --config lora_config.yaml
```

### What to watch during training

Training prints loss every `steps_per_report` steps and validation loss every `steps_per_eval` steps. The two numbers that matter:

- **Training loss**: Should decrease steadily. If it plateaus early, the model may need more capacity (higher rank) or a different learning rate.
- **Validation loss**: Should decrease roughly in parallel with training loss. If training loss keeps dropping but validation loss starts rising, you're **overfitting**.

```
Iter 100: Train loss 1.823, Val loss 1.901    ← healthy, both decreasing
Iter 500: Train loss 0.412, Val loss 0.834    ← gap widening, early overfitting
Iter 800: Train loss 0.201, Val loss 1.102    ← overfitting, should have stopped earlier
```

---

## Step 3: Evaluate properly

### Quantitative: test set perplexity

```bash
mlx_lm.lora \
    --model mlx-community/Phi-4-mini-instruct-4bit \
    --adapter-path ./adapters \
    --data ./data \
    --test
```

This gives you test loss / perplexity. Lower is better, but **this number alone doesn't tell you if the outputs are good**. A model can have low perplexity and still produce badly formatted messages.

### Qualitative: generate on real diffs

This is the evaluation that actually matters. Test on 5-10 diverse diffs and check the outputs manually:

```bash
mlx_lm.generate \
    --model mlx-community/Phi-4-mini-instruct-4bit \
    --adapter-path ./adapters \
    --max-tokens 100 \
    --prompt "YOUR_PROMPT_WITH_DIFF_HERE"
```

Or serve the model and test via curl (matches the real inference path):

```bash
# Terminal 1: serve
mlx_lm.server \
    --model mlx-community/Phi-4-mini-instruct-4bit \
    --adapter-path ./adapters \
    --port 8080

# Terminal 2: test
curl -s http://localhost:8080/v1/chat/completions \
    -H "Content-Type: application/json" \
    -d '{
        "messages": [
            {"role": "system", "content": "You are a commit message generator..."},
            {"role": "user", "content": "Files changed:\n main.go | 5 +++--\n\nDiff:\ndiff --git a/main.go..."}
        ],
        "temperature": 0.3,
        "max_completion_tokens": 100
    }' | python3 -c "import sys,json; print(json.load(sys.stdin)['choices'][0]['message']['content'])"
```

### What to look for in outputs

Score each output on these criteria:
1. **Format**: Does it start with `type(scope):` or `type:`? Is it lowercase?
2. **Relevance**: Does it describe what the diff actually does?
3. **Conciseness**: Is it under 72 characters? Does it avoid repeating the diff?
4. **Noise**: Does it include markdown, quotes, explanations, or preamble?

If format is wrong but relevance is good → data quality problem (go back to step 1).
If format is right but relevance is poor → need more/better training data or more iterations.
If it's producing noise/preamble → try `--mask-prompt` (see below).

---

## Step 4: Diagnose and adjust

### Problem: Model doesn't follow conventional commit format

**Cause**: Training data contains non-conventional messages.
**Fix**: Re-generate data using `prepare-training` which applies `ValidateMessage`. This is the most common issue and almost certainly your current problem.

### Problem: Model overfits (low train loss, high val loss)

**Fixes** (try one at a time):
- Reduce `iters` — use the checkpoint from before val loss started rising
- Increase `dropout` (try 0.1)
- Reduce `rank` (try 4)
- Reduce `num_layers` (try 8)
- Add more training data

### Problem: Model underfits (high train loss, doesn't converge)

**Fixes**:
- Increase `iters` (try 2000-3000)
- Increase `learning_rate` (try 5e-4, but watch for instability)
- Increase `rank` (try 16)
- Enable cosine decay LR schedule with warmup:
  ```yaml
  lr_schedule:
    name: cosine_decay
    warmup: 100
    warmup_init: 1e-7
    arguments: [1e-5, 2000, 1e-7]
  ```

### Problem: Model outputs noise, explanations, or markdown around the commit message

**Fixes**:
- Add `mask_prompt: true` to your config. This computes loss only on the assistant's response (the commit message), not the system prompt or diff. It focuses learning on message generation rather than diff reconstruction.
- Make sure the system prompt says "Output the raw commit message only. No markdown, no code fences, no quotes, no explanation."
- At inference time, use `temperature: 0.3` or lower and `max_completion_tokens: 100`.

### Problem: Model produces generic/vague messages

**Fixes**:
- Use more training data (scale from 5K to 20K+)
- Make sure training data has diverse examples across different commit types
- Ensure the diff preprocessing in training matches inference (use `prepare-training`, not manual conversion)

---

## Step 5: Hyperparameter tuning order

Once your data is clean, tune hyperparameters in this order of impact:

1. **Data size and quality** — Always the highest-leverage change
2. **Number of iterations** — Find the sweet spot before overfitting (use val loss)
3. **`mask_prompt`** — Try `true` vs `false`; for this task, `true` often helps
4. **Learning rate** — Try 1e-4, 2e-4, 5e-4
5. **LoRA rank** — Try 4, 8, 16
6. **Learning rate schedule** — Cosine decay with warmup can help convergence
7. **`num_layers`** — 8 vs 16 (fewer layers = less capacity but less overfitting)
8. **`scale`** — Try 8.0, 10.0, 16.0 (controls the magnitude of LoRA updates)

For each change, train and evaluate. Keep a simple log:

```
Run | Data  | Iters | LR    | Rank | Mask | Val Loss | Format% | Notes
1   | 10K   | 1000  | 2e-4  | 8    | no   | 1.23     | 40%     | baseline
2   | 10K*  | 1000  | 2e-4  | 8    | no   | 0.89     | 95%     | filtered data
3   | 10K*  | 1000  | 2e-4  | 8    | yes  | 0.91     | 97%     | mask_prompt helped
4   | 20K*  | 1500  | 2e-4  | 8    | yes  | 0.82     | 98%     | more data
```

\* = data generated with `prepare-training` (filtered)

---

## Recommended next steps for your current situation

Based on where you are (trained on unfiltered data, model generates messages but wrong format):

1. **Re-generate training data** with `prepare-training` to filter for conventional commits only
2. **Train again** with the same config — just clean data will likely fix the format problem
3. **Add `mask_prompt: true`** to your config for the next run
4. **Evaluate** on 10 real diffs and check format compliance
5. **Scale data** to 20-30K if quality looks good but you want more consistency

---

## Further reading

- **MLX-LM documentation**: https://github.com/ml-explore/mlx-lm — The README and examples/ directory cover all CLI commands and config options
- **QLoRA paper** (Dettmers et al., 2023): https://arxiv.org/abs/2305.14314 — Explains the theory behind quantized fine-tuning; useful for understanding why rank, scale, and dropout matter
- **LoRA paper** (Hu et al., 2021): https://arxiv.org/abs/2106.09685 — The original Low-Rank Adaptation paper; short and readable
- **Hugging Face fine-tuning guide**: https://huggingface.co/docs/transformers/en/training — Not MLX-specific, but the concepts (data prep, evaluation, overfitting) are universal
- **Sebastian Raschka's LLM fine-tuning guide**: https://magazine.sebastianraschka.com/p/practical-tips-for-finetuning-llms — Practical tips from a researcher; covers data quality, hyperparameter selection, and common pitfalls
- **Conventional Commits spec**: https://www.conventionalcommits.org/ — The format standard your training data should follow
- **CommitBench dataset**: https://huggingface.co/datasets/Maxscha/commitbench — The source dataset; understanding its structure helps with data filtering decisions
