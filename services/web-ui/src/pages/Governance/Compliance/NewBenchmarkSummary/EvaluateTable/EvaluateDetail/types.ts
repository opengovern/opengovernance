export interface RunDetail {
    es_id:              string;
    es_index:           string;
    controls:           Controls;
    compliance_summary: ComplianceSummary;
    job_summary:        JobSummary;
}

export interface ComplianceSummary {
    alarm: number;
}

export interface Controls {
    [key: string]: AzureUseByokForStorageAccountEncryption
}

export interface AzureUseByokForStorageAccountEncryption {
    severity: string;
    alarms:   number;
    oks:      number;
}

export interface JobSummary {
    job_id:          number;
    auditable:       boolean;
    framework_id:    string;
    job_started_at:  Date;
    integration_ids: null;
}
