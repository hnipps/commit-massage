import json

from datasets import load_dataset


def split_dataset():

    ds = load_dataset("Maxscha/commitbench")

    for split, path in [
        ("train", "raw_train.jsonl"),
        ("validation", "raw_valid.jsonl"),
        ("test", "raw_test.jsonl"),
    ]:
        with open(path, "w") as f:
            for row in ds[split]:
                f.write(
                    json.dumps({"diff": row["diff"], "message": row["message"]}) + "\n"
                )


if __name__ == "__main__":
    split_dataset()
