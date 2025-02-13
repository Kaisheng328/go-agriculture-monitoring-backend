from flask import Flask, request, jsonify
import joblib
import numpy as np

# Load the trained model
model = joblib.load("soil_moisture_model.pkl")

app = Flask(__name__)

@app.route("/predict", methods=["POST"])
def predict():
    data = request.json
    if "temperature" not in data or "humidity" not in data:
        return jsonify({"error": "Missing temperature or humidity"}), 400

    # Prepare input for the model
    features = np.array([[data["temperature"], data["humidity"]]])
    prediction = model.predict(features)[0]  # Get predicted soil moisture

    return jsonify({"predicted_soil_moisture": round(prediction, 2)})

if __name__ == "__main__":
    app.run(host="0.0.0.0", port=5001)  # Run API on port 5001
