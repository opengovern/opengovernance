// @ts-nocheck
import Wizard from '@cloudscape-design/components/wizard'
import {
    Box,
    Container,
    FormField,
    Header,
    Input,
    KeyValuePairs,
    Link,
    ProgressBar,
    SpaceBetween,
    Spinner,
} from '@cloudscape-design/components'
import { Button, Card, Divider, Flex, Text } from '@tremor/react'
import './style.css'
import License from './stepComponents/step1'
import { useEffect, useState } from 'react'
import Integrations from './stepComponents/step3'
import Complete from './stepComponents/step4'
import UserCredinations from './stepComponents/step2'
import { CheckDetail, CheckResponse, SetupRes, WizarData } from './types'
import { notificationAtom } from '../../store'
import { useSetAtom } from 'jotai'
import axios from 'axios'
export default function SetupWizard() {
    const [activeStepIndex, setActiveStepIndex] = useState<number>(0)
    const [loading, setLoading] = useState<boolean>(true)
    const [wizardLoading, setWizardLoading] = useState<boolean>(false)
    const [progressLoading, setProgressLoading] = useState<boolean>(false)
    const [awsProgress, setAwsProgress] = useState()
    const [azureProgress, setAzureProgress] = useState()
    const [sampleProgress, setSampleProgress] = useState()
    const [migrationsProgress, setMigrationsProgress] = useState()

    const [wizardData, setWizardData] = useState<WizarData>()
    const [start, setStart] = useState<boolean>(false)
    const [setup, setSetup] = useState<boolean>(false)
    const [check, setCheck] = useState<boolean>(false)
    const [checkRes, setCheckRes] = useState<CheckResponse[]>()
    const [setupRes, setSetupRes] = useState<SetupRes>()

    const setNotification = useSetAtom(notificationAtom)

    const OnClickNext = (index: number) => {
        if (index == 1 || index == 0) {
            setActiveStepIndex(index)
            return
        }
        if (index === 2) {
            if (wizardData?.userData?.email && wizardData?.userData?.password) {
                setActiveStepIndex(index)
                return
            } else {
                setNotification({
                    text: 'Please enter your email and password',
                    type: 'error',
                })
                return
            }
        }
        if (index === 3) {
            if (wizardData?.sampleLoaded === 'sample') {
                setActiveStepIndex(index)
                return
            } else {
                console.log(wizardData)
                if (!wizardData?.awsData) {
                    setNotification({
                        text: 'Please  AWS and complete the form',
                        type: 'error',
                    })
                    return
                }
                if (!wizardData?.azureData) {
                    setNotification({
                        text: 'Please Select an Azure and complete the form',
                        type: 'error',
                    })
                    return
                }
                if (
                    !wizardData?.azureData?.applicationId ||
                    !wizardData?.azureData?.directoryId ||
                    !wizardData?.azureData?.objectId ||
                    !wizardData?.azureData?.secretValue
                ) {
                    setNotification({
                        text: 'Please Complete the Azure form corrctly',
                        type: 'error',
                    })
                    return
                }
                if (
                    !wizardData?.awsData?.accessKey ||
                    !wizardData?.awsData?.accessSecret ||
                    !wizardData?.awsData?.role
                ) {
                    setNotification({
                        text: 'Please Complete the AWS form corrctly',
                        type: 'error',
                    })
                    return
                }
                setActiveStepIndex(index)
            }
        }
    }
    const OnSubmitData = () => {
        let url = ''
        if (window.location.origin === 'http://localhost:3000') {
            url = window.__RUNTIME_CONFIG__.REACT_APP_BASE_URL
        } else {
            url = window.location.origin
        }
        if (wizardData?.sampleLoaded === 'manual') {
            setWizardLoading(true)
            const body = {
                aws_credentials: {
                    accessKey: wizardData?.awsData?.accessKey,
                    secretKey: wizardData?.awsData?.accessSecret,
                    assumeRoleName: wizardData?.awsData?.role,
                },
                azure_credentials: {
                    tenantId: wizardData?.azureData?.directoryId,
                    objectId: wizardData?.azureData?.objectId,
                    clientSecret: wizardData?.azureData?.secretValue,
                    clientId: wizardData?.azureData?.applicationId,
                },
            }
            // @ts-ignore
            const token = JSON.parse(localStorage.getItem('kaytu_auth')).token

            const config = {
                headers: {
                    Authorization: `Bearer ${token}`,
                },
            }
            axios
                .post(`${url}/kaytu/auth/api/v3/setup/check`, body, config)
                .then((res: CheckResponse) => {
                    setWizardLoading(false)
                    setCheck(true)
                    setCheckRes(res.data)
                    if (!HasError(res.data)) {
                        Setup()
                    }
                })
                .catch((err) => {
                    setWizardLoading(false)

                    setNotification({
                        text: 'Can not Connect to the server',
                        type: 'error',
                    })
                })
        } else {
            setCheck(true)
            Setup()
        }
    }
    const HasError = (data: CheckResponse[]) => {
        let flag = false
        data.map((item) => {
            if (item.summary.failedChecks > 0) {
                flag = true
            }
        })
        return flag
    }
    const Setup = () => {
        let url = ''
        if (window.location.origin === 'http://localhost:3000') {
            url = window.__RUNTIME_CONFIG__.REACT_APP_BASE_URL
        } else {
            url = window.location.origin
        }
        setWizardLoading(true)

        const body = {
            aws_credentials: {
                accessKey: wizardData?.awsData?.accessKey,
                secretKey: wizardData?.awsData?.accessSecret,
                assumeRoleName: wizardData?.awsData?.role,
            },
            azure_credentials: {
                tenantId: wizardData?.azureData?.directoryId,
                objectId: wizardData?.azureData?.objectId,
                clientSecret: wizardData?.azureData?.secretValue,
                clientId: wizardData?.azureData?.applicationId,
            },
            include_sample_data:
                wizardData?.sampleLoaded === 'sample' ? true : false,
            create_user: {
                email_address: wizardData?.userData?.email,
                password: wizardData?.userData?.password,
            },
        }
        // @ts-ignore
        const token = JSON.parse(localStorage.getItem('kaytu_auth')).token

        const config = {
            headers: {
                Authorization: `Bearer ${token}`,
            },
        }
        axios
            .post(`${url}/kaytu/auth/api/v3/setup`, body, config)
            .then((res: SetupRes) => {
                setWizardLoading(false)
                setSetup(true)
                setSetupRes(res.data)
                GetMigration()
                if (wizardData?.sampleLoaded === 'sample') {
                    GetSampleDataStatus()
                } else {
                    GetProgress(res.data.aws_trigger_id, 'aws')
                    GetProgress(res.data.azure_trigger_id, 'azure')
                }
            })
            .catch((err) => {
                setWizardLoading(false)
                if (err?.response?.data?.message) {
                    setNotification({
                        text: err?.response?.data?.message,
                        type: 'error',
                    })
                } else {
                    setNotification({
                        text: 'Can not Connect to the server',
                        type: 'error',
                    })
                }
            })
    }
    const GetProgress = (id: string, type: string) => {
        let url = ''
        if (window.location.origin === 'http://localhost:3000') {
            url = window.__RUNTIME_CONFIG__.REACT_APP_BASE_URL
        } else {
            url = window.location.origin
        }
        setProgressLoading(true)

        const body = {
            trigger_id: id,
        }
        // @ts-ignore
        const token = JSON.parse(localStorage.getItem('kaytu_auth')).token

        const config = {
            headers: {
                Authorization: `Bearer ${token}`,
            },
        }
        axios
            .post(`${url}/kaytu/schedule/api/v3/discovery/status`, body, config)
            .then((res: SetupRes) => {
                setProgressLoading(false)
                if (type == 'aws') {
                    setAwsProgress(res.data)
                } else {
                    setAzureProgress(res.data)
                }
            })
            .catch((err) => {
                setProgressLoading(false)
                setNotification({
                    text: 'Can not Connect to the server',
                    type: 'error',
                })
            })
    }
    const GetMigration = () => {
        let url = ''
        if (window.location.origin === 'http://localhost:3000') {
            url = window.__RUNTIME_CONFIG__.REACT_APP_BASE_URL
        } else {
            url = window.location.origin
        }
        setProgressLoading(true)

        // @ts-ignore
        const token = JSON.parse(localStorage.getItem('kaytu_auth')).token

        const config = {
            headers: {
                Authorization: `Bearer ${token}`,
            },
        }
        axios
            .get(`${url}/kaytu/workspace/api/v3/sample/sync/status`, config)
            .then((res: SetupRes) => {
                setProgressLoading(false)
                setMigrationsProgress(res.data)
            })
            .catch((err) => {
                setProgressLoading(false)
                setNotification({
                    text: 'Can not Connect to the server',
                    type: 'error',
                })
            })
    }
    const GetSampleDataStatus = () => {
        let url = ''
        if (window.location.origin === 'http://localhost:3000') {
            url = window.__RUNTIME_CONFIG__.REACT_APP_BASE_URL
        } else {
            url = window.location.origin
        }
        setProgressLoading(true)

        // @ts-ignore
        const token = JSON.parse(localStorage.getItem('kaytu_auth')).token

        const config = {
            headers: {
                Authorization: `Bearer ${token}`,
            },
        }
        axios
            .get(`${url}/kaytu/workspace/api/v3/migration/status`, config)
            .then((res: SetupRes) => {
                setProgressLoading(false)
                setSampleProgress(res.data)
            })
            .catch((err) => {
                console.log(err)
                setProgressLoading(false)
                setNotification({
                    text: err.message,
                    type: 'error',
                })
            })
    }
    useEffect(()=>{
        setCheck(false)
    },[activeStepIndex])
    return (
        <>
            <Flex
                justifyContent="center"
                alignItems="center"
                className="w-full"
            >
                <Card className="flex justify-center items-center wizard-card w-4/5 flex-col">
                    {setup ? (
                        <>
                            <Header>Setup Progress</Header>
                            <SpaceBetween size="xxl" className="w-1/2 mt-2">
                                <SpaceBetween
                                    size="xs"
                                    className="step-1-review"
                                >
                                    <SpaceBetween size="l">
                                        <Container
                                            header={
                                                <Header
                                                    variant="h2"
                                                    headingTagOverride="h3"
                                                >
                                                    Creating Login
                                                </Header>
                                            }
                                        >
                                            <ProgressBar
                                                value={migrationsProgress}
                                                status={
                                                    setupRes?.created_user
                                                        ? 'success'
                                                        : 'error'
                                                }
                                                resultText="Created"
                                            />
                                        </Container>
                                    </SpaceBetween>
                                </SpaceBetween>
                                <SpaceBetween
                                    size="xs"
                                    className="step-1-review"
                                >
                                    <SpaceBetween size="l">
                                        <Container
                                            header={
                                                <Header
                                                    variant="h2"
                                                    headingTagOverride="h3"
                                                >
                                                    Setting up Metadata
                                                </Header>
                                            }
                                        >
                                            <ProgressBar
                                                value={50}
                                                status={
                                                    migrationsProgress ==
                                                    'COMPLETED'
                                                        ? 'success'
                                                        : migrationsProgress ==
                                                          'FAILED'
                                                        ? 'error'
                                                        : 'in-progress'
                                                }
                                                resultText="Completed"
                                            />
                                            {/* <ProgressBar
                                            value={migrationsProgress}
                                        /> */}
                                        </Container>
                                    </SpaceBetween>
                                </SpaceBetween>
                                {wizardData?.sampleLoaded === 'manual' && (
                                    <>
                                        <SpaceBetween
                                            size="xs"
                                            className="step-1-review"
                                        >
                                            <SpaceBetween size="l">
                                                <Container
                                                    header={
                                                        <Header
                                                            variant="h2"
                                                            headingTagOverride="h3"
                                                        >
                                                            Setting up AWS
                                                            Integration
                                                        </Header>
                                                    }
                                                >
                                                    <ProgressBar
                                                        value={
                                                            awsProgress
                                                                ?.trigger_id_progress_summary
                                                                .processed_count /
                                                            awsProgress
                                                                ?.trigger_id_progress_summary
                                                                .total_count
                                                        }
                                                    />
                                                </Container>
                                            </SpaceBetween>
                                        </SpaceBetween>
                                        <SpaceBetween
                                            size="xs"
                                            className="step-1-review"
                                        >
                                            <SpaceBetween size="l">
                                                <Container
                                                    header={
                                                        <Header
                                                            variant="h2"
                                                            headingTagOverride="h3"
                                                        >
                                                            Setting up Azure
                                                            Integration
                                                        </Header>
                                                    }
                                                >
                                                    {/* add status and value */}
                                                    <ProgressBar
                                                        value={
                                                            azureProgress
                                                                ?.trigger_id_progress_summary
                                                                .processed_count /
                                                            azureProgress
                                                                ?.trigger_id_progress_summary
                                                                .total_count
                                                        }
                                                    />
                                                </Container>
                                            </SpaceBetween>
                                        </SpaceBetween>
                                    </>
                                )}

                                {wizardData?.sampleLoaded == 'sample' && (
                                    <>
                                        <SpaceBetween
                                            size="xs"
                                            className="step-1-review"
                                        >
                                            <SpaceBetween size="l">
                                                <Container
                                                    header={
                                                        <Header
                                                            variant="h2"
                                                            headingTagOverride="h3"
                                                        >
                                                            Loading Sample Data
                                                        </Header>
                                                    }
                                                >
                                                    Status : {sampleProgress}
                                                    {/* <ProgressBar
                                                    value={sampleProgress}
                                                /> */}
                                                </Container>
                                            </SpaceBetween>
                                        </SpaceBetween>
                                    </>
                                )}
                            </SpaceBetween>
                        </>
                    ) : (
                        <>
                            {' '}
                            {start ? (
                                <>
                                    {wizardLoading ? (
                                        <>
                                            <Spinner />
                                        </>
                                    ) : (
                                        <>
                                            <Wizard
                                                className="w-full"
                                                i18nStrings={{
                                                    stepNumberLabel: (
                                                        stepNumber
                                                    ) => `Step ${stepNumber}`,
                                                    collapsedStepsLabel: (
                                                        stepNumber,
                                                        stepsCount
                                                    ) =>
                                                        `Step ${stepNumber} of ${stepsCount}`,
                                                    skipToButtonLabel: (
                                                        step,
                                                        stepNumber
                                                    ) =>
                                                        `Skip to ${step.title}`,
                                                    navigationAriaLabel:
                                                        'Steps',
                                                    cancelButton: '',
                                                    previousButton: 'Previous',
                                                    nextButton: 'Next',
                                                    submitButton: 'Submit',
                                                    optional: 'optional',
                                                }}
                                                onNavigate={({ detail }) =>
                                                    OnClickNext(
                                                        detail.requestedStepIndex
                                                    )
                                                }
                                                activeStepIndex={
                                                    activeStepIndex
                                                }
                                                steps={[
                                                    {
                                                        title: 'License Agreement',

                                                        content: (
                                                            <License
                                                                setLoading={
                                                                    setLoading
                                                                }
                                                            />
                                                        ),
                                                    },
                                                    {
                                                        title: 'Setup Login',

                                                        content: (
                                                            <UserCredinations
                                                                setLoading={
                                                                    setLoading
                                                                }
                                                                wizardData={
                                                                    wizardData
                                                                }
                                                                setWizardData={
                                                                    setWizardData
                                                                }
                                                            />
                                                        ),
                                                    },
                                                    {
                                                        title: 'Integrations',
                                                        content: (
                                                            <Integrations
                                                                setLoading={
                                                                    setLoading
                                                                }
                                                                wizardData={
                                                                    wizardData
                                                                }
                                                                setWizardData={
                                                                    setWizardData
                                                                }
                                                            />
                                                        ),
                                                        isOptional: false,
                                                    },
                                                    {
                                                        title: 'Review and create',
                                                        content: (
                                                            <Complete
                                                                setLoading={
                                                                    setLoading
                                                                }
                                                                setActiveStepIndex={
                                                                    setActiveStepIndex
                                                                }
                                                                wizardData={
                                                                    wizardData
                                                                }
                                                                setWizardData={
                                                                    setWizardData
                                                                }
                                                            />
                                                        ),
                                                        isOptional: false,
                                                    },
                                                ]}
                                                isLoadingNextStep={loading}
                                                onSubmit={() => OnSubmitData()}
                                            />
                                            {check &&
                                                wizardData?.sampleLoaded ===
                                                    'manual' &&
                                                HasError(checkRes) && (
                                                    <>
                                                        <Card className="mt-3">
                                                            <>
                                                                {checkRes?.map(
                                                                    (item) => {
                                                                        return (
                                                                            <>
                                                                                <Text>
                                                                                    {item.checkDetails.map(
                                                                                        (
                                                                                            detail
                                                                                        ) => {
                                                                                            return (
                                                                                                <>
                                                                                                    {detail.status ==
                                                                                                        'error' && (
                                                                                                        <>
                                                                                                            {
                                                                                                                detail.message
                                                                                                            }
                                                                                                            <Divider />
                                                                                                        </>
                                                                                                    )}
                                                                                                </>
                                                                                            )
                                                                                        }
                                                                                    )}
                                                                                </Text>
                                                                            </>
                                                                        )
                                                                    }
                                                                )}
                                                            </>
                                                        </Card>
                                                    </>
                                                )}
                                        </>
                                    )}
                                </>
                            ) : (
                                <>
                                    <Flex
                                        className="w-full min-h-48"
                                        justifyContent="center"
                                        alignItems="center"
                                    >
                                        <Button
                                            onClick={() => {
                                                setStart(true)
                                            }}
                                            className=" w-1/2 text-3xl"
                                        >
                                            Start Setup
                                        </Button>
                                    </Flex>
                                </>
                            )}
                        </>
                    )}
                </Card>
            </Flex>
        </>
    )
}
