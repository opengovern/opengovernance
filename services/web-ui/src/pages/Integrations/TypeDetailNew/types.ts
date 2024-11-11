export interface Integration {
    integration_id:   string;
    provider_id:      string;
    name:             string;
    integration_type: string;
    annotations:      Annotations;
    labels:           Annotations;
    credential_id:    string;
    state:            string;
    last_check:       Date;
}

export interface Annotations {
}

export interface Schema {
    integration_type_id:    string;
    integration_name:       string;
    help_text_md:           string;
    platform_documentation: string;
    provider_documentation: string;
    icon:                   string;
    discover:               Discover;
    list:                   List;
    view:                   View;
    actions:                Actions;
}

export interface Actions {
    credentials:  Credential[];
    integrations: Integration[];
}

export interface Credential {
    type:            string;
    label:           string;
    editableFields?: string[];
    confirm?:        CredentialConfirm;
}

export interface CredentialConfirm {
    message:   string;
    condition: Condition;
}

export interface Condition {
    field:        string;
    operator:     string;
    value:        number;
    errorMessage: string;
}

export interface Integration {
    type:            string;
    label:           string;
    editableFields?: string[];
    confirm?:        IntegrationConfirm;
    tooltip?:        string;
}

export interface IntegrationConfirm {
    message: string;
}

export interface Discover {
    description:      string;
    credentialInputs: CredentialInput[];
}

export interface CredentialInput {
    type:     string;
    label:    string;
    priority: number;
    fields:   CredentialInputField[];
}

export interface CredentialInputField {
    name:               string;
    label:              string;
    inputType:          string;
    required:           boolean;
    order:              number;
    validation:         Validation;
    info:               string;
    external_help_url?: string;
    conditional?:       Conditional;
}

export interface Conditional {
    field:     string;
    isPresent: boolean;
}

export interface Validation {
    pattern?:       string;
    errorMessage:   string;
    minLength?:     number;
    maxLength?:     number;
    fileTypes?:     string[];
    maxFileSizeMB?: number;
}

export interface List {
    credentials:  Credentials;
    integrations: Integrations;
}

export interface Credentials {
    defaultPageSize: number;
    display:         CredentialsDisplay;
}

export interface CredentialsDisplay {
    displayFields:  DisplayField[];
   
}

export interface DisplayField {
    name:           string;
    label:          string;
    fieldType:      string;
    order:          number;
    sortable:       boolean;
    filterable:     boolean;
    info:           string;
    statusOptions?: StatusOption[];
}

export interface StatusOption {
    value: string;
    label: string;
    color: string;
}

export interface Integrations {
    defaultPageSize: number;
    display:         IntegrationsDisplay;
}

export interface IntegrationsDisplay {
    id:             string;
    name:           string;
    description:    string;
    logo:           string;
    help_text:      string;
    displayFields:  DisplayField[];
   
}

export interface View {
    integration_details: Details;
    credential_details:  Details;
}

export interface Details {
    description:    string;
    fields:         CredentialDetailsField[];
  
}

export interface CredentialDetailsField {
    name:           string;
    label:          string;
    fieldType:      string;
    order:          number;
    info:           string;
    conditional?:   Conditional;
    statusOptions?: StatusOption[];
    
}

export interface Credentials {
    id:               string;
    secret:           string;
    integration_type: string;
    metadata:         Metadata;
    created_at:       Date;
    updated_at:       Date;
}

export interface Metadata {
}
