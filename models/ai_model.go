package models

type TrainModelRequest struct {
	PlantName string `json:"plant_name" binding:"required"`
}

// TrainModelResponse represents the response from Python training service
type TrainModelResponse struct {
	Success              bool                   `json:"success"`
	Message              string                 `json:"message"`
	ModelPath            string                 `json:"model_path,omitempty"`
	R2Score              float64                `json:"r2_score,omitempty"`
	RMSE                 float64                `json:"rmse,omitempty"`
	MAE                  float64                `json:"mae,omitempty"`
	BestModel            string                 `json:"best_model,omitempty"`
	TrainingTime         string                 `json:"training_time,omitempty"`
	TrainingDurationSecs float64                `json:"training_duration_seconds,omitempty"`
	CreatedAt            string                 `json:"created_at,omitempty"`
	CompletedAt          string                 `json:"completed_at,omitempty"`
	CreatedReadable      string                 `json:"created_readable,omitempty"`
	DataPoints           int                    `json:"data_points,omitempty"`
	OriginalDataPoints   int                    `json:"original_data_points,omitempty"`
	AllResults           map[string]interface{} `json:"all_results,omitempty"`
}
