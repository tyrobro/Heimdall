import os
import asyncio
import pandas as pd
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
from sklearn.ensemble import RandomForestClassifier
import pickle

app = FastAPI(title="Heimdall ML Classifier (Batch Learning)")

model = None
last_trained_size = 0
BATCH_THRESHOLD = 100


def extract_features(magic_bytes_hex: str, size_mb: float):
    try:
        hex_hash = hash(magic_bytes_hex) % 10000
    except:
        hex_hash = 0
    return [[hex_hash, size_mb]]


def retrain_model_from_telemetry():
    global model, last_trained_size
    telemetry_path = "../telemetry.csv"

    if not os.path.exists(telemetry_path):
        print("Waiting for telemetry data...")
        return False

    try:
        df = pd.read_csv(telemetry_path)
        current_size = len(df)

        if current_size - last_trained_size >= BATCH_THRESHOLD:
            print(
                f"Threshold reached ({current_size} total events). Initiating Batch Retraining..."
            )

            action_counts = (
                df.groupby(["file_name", "action"]).size().unstack(fill_value=0)
            )

            X = []
            y = []

            for index, row in action_counts.iterrows():
                fake_magic = index.split(".")[-1] if "." in index else "unknown"
                fake_size = len(index) * 2.5

                features = extract_features(fake_magic, fake_size)[0]
                is_read_heavy = 1 if row.get("READ", 0) > row.get("WRITE", 0) else 0

                X.append(features)
                y.append(is_read_heavy)

            if len(X) > 0:
                new_model = RandomForestClassifier(n_estimators=50, random_state=42)
                new_model.fit(X, y)
                model = new_model
                last_trained_size = current_size

                with open("heimdall_model.pkl", "wb") as f:
                    pickle.dump(model, f)

                print(
                    f"Batch Retraining Complete! Brain updated at {current_size} events."
                )
                return True

    except Exception as e:
        print(f"Retraining failed: {e}")

    return False


async def watch_telemetry_loop():
    while True:
        await asyncio.sleep(10)
        retrain_model_from_telemetry()


@app.on_event("startup")
async def startup_event():
    global model
    if os.path.exists("heimdall_model.pkl"):
        with open("heimdall_model.pkl", "rb") as f:
            model = pickle.load(f)
        print("Loaded existing brain from disk.")
    else:
        print(
            "No brain found. Waiting for Gateway telemetry to initiate first training loop..."
        )

    asyncio.create_task(watch_telemetry_loop())


class FileData(BaseModel):
    file_name: str
    magic_bytes: str


@app.post("/predict")
async def predict_workload(data: FileData):
    if model is None:
        return {
            "file_name": data.file_name,
            "prediction": "Unknown (Training in progress)",
        }

    try:
        size_mb = 10.0

        features = extract_features(data.magic_bytes, size_mb)

        prediction_array = model.predict(features)
        is_read_heavy = prediction_array[0]

        result = "Read-Heavy" if is_read_heavy == 1 else "Write-Heavy"

        return {"file_name": data.file_name, "prediction": result}

    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host="0.0.0.0", port=8000)
