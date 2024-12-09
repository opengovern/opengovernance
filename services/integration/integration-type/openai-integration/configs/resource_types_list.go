package configs

var TablesToResourceTypes = map[string]string{
	"openai_project":                 "OpenAI/Project",
	"openai_project_api_key":         "OpenAI/Project/ApiKey",
	"openai_project_rate_limit":      "OpenAI/Project/RateLimit",
	"openai_project_service_account": "OpenAI/Project/ServiceAccount",
	"openai_project_user":            "OpenAI/Project/User",
	"openai_model":                   "OpenAI/Model",
	"openai_file":                    "OpenAI/File",
	"openai_vector_store":            "OpenAI/VectorStore",
	"openai_assistant":               "OpenAI/Assistant",
}

var ResourceTypesList = []string{
	//"OpenAI/Project",
	//"OpenAI/Project/ApiKey",
	//"OpenAI/Project/RateLimit",
	//"OpenAI/Project/ServiceAccount",
	//"OpenAI/Project/User",
	"OpenAI/Model",
	"OpenAI/File",
	"OpenAI/VectorStore",
	"OpenAI/Assistant",
}
