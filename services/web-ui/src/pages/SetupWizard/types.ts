export interface WizarData {
    azureData?: {
        applicationId?: string
        objectId?: string
        directoryId?: string
        secretValue?: string
    }
    awsData?: {
        accessKey?: string
        accessSecret?: string
        role?: string
    }
    sampleLoaded?: string
    userData?: {
        email?: string
        password?: string
    }
}
export interface CheckResponse {
    provider: string
    summary: Summary
    checkDetails: CheckDetail[]
}

export interface CheckDetail {
    provider: string
    checkType: string
    status: string
    message: string
    subscriptions?: string[]
}

export interface Summary {
    totalChecks: number
    successfulChecks: number
    failedChecks: number
}

export interface SetupRes {
    created_user : boolean
    metadata : string
    sample_data_import :boolean
    aws_trigger_id: string
    azure_trigger_id : string
}