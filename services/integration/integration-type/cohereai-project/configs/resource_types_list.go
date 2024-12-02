package configs

var TablesToResourceTypes = map[string]string{
	 "cohereai_connectors": "CohereAI/Connectors",
  "cohereai_models": "CohereAI/Models",
  "cohereai_datasets": "CohereAI/Datasets",
  "cohereai_fine_tuned_models": "CohereAI/FineTunedModel",
  "cohereai_embed_jobs": "CohereAI/EmbedJob",
}

var ResourceTypesList = []string{
  "CohereAI/Connectors",
  "CohereAI/Models",
  "CohereAI/Datasets",
  "CohereAI/FineTunedModel",
  "CohereAI/EmbedJob",
}