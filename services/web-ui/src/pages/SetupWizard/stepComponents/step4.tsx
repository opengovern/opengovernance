import {
    ArrowTopRightOnSquareIcon,
    BanknotesIcon,
    ChevronRightIcon,
    CubeIcon,
    CursorArrowRaysIcon,
    PuzzlePieceIcon,
    ShieldCheckIcon,
} from '@heroicons/react/24/outline'
import { Card, Flex, Grid, Icon, Text, Title } from '@tremor/react'
import { useNavigate, useParams } from 'react-router-dom'
import Check from '../../../icons/Check.svg'
import User from '../../../icons/User.svg'
import Dollar from '../../../icons/Dollar.svg'
import Cable from '../../../icons/Cable.svg'
import Cube from '../../../icons/Cube.svg'
import Checkbox from '@cloudscape-design/components/checkbox'
import {
    Box,
    Button,
    Container,
    ExpandableSection,
    Header,
    SpaceBetween,
    KeyValuePairs,
} from '@cloudscape-design/components'
import ProgressBar from '@cloudscape-design/components/progress-bar'
import { link } from 'fs'
import { useEffect, useState } from 'react'
import axios from 'axios'
import ReactMarkdown from 'react-markdown'
import { WizarData } from '../types'
interface Props {
    setLoading: Function
    setActiveStepIndex: Function
    wizardData: WizarData
    setWizardData: Function
}

export default function Complete({ setLoading, setActiveStepIndex,wizardData,setWizardData }: Props) {
    const workspace = useParams<{ ws: string }>().ws
    const navigate = useNavigate()
    const [checked, setChecked] = useState(false)
    const [markdown, setMarkdown] = useState('')

    return (
        <Box margin={{ bottom: 'l' }}>
            <SpaceBetween size="xxl">
                <SpaceBetween size="xs" className="step-1-review">
                    <Header
                        variant="h3"
                        headingTagOverride="h2"
                        actions={
                            <Button
                                className="edit-step-btn"
                                onClick={() => setActiveStepIndex(1)}
                            >
                                Edit
                            </Button>
                        }
                    >
                        Login Credentials
                    </Header>
                    <SpaceBetween size="l">
                        <Container
                            header={
                                <Header variant="h2" headingTagOverride="h3">
                                    Credential for Login
                                </Header>
                            }
                        >
                            <KeyValuePairs
                                columns={2}
                                items={[
                                    {
                                        label: 'Email',
                                        value: wizardData?.userData?.email,
                                    },
                                    {
                                        label: 'Password',
                                        value: wizardData?.userData?.password,
                                    },
                                ]}
                            />
                        </Container>
                    </SpaceBetween>
                </SpaceBetween>

                <SpaceBetween size="xs" className="step-3-review">
                    <Header
                        variant="h3"
                        headingTagOverride="h2"
                        actions={
                            <Button
                                className="edit-step-btn"
                                onClick={() => setActiveStepIndex(2)}
                            >
                                Edit
                            </Button>
                        }
                    >
                        Integrations
                    </Header>
                    <SpaceBetween size="l">
                        {wizardData?.sampleLoaded === 'sample' && (
                            <>
                                <Container
                                    header={
                                        <Header
                                            variant="h2"
                                            headingTagOverride="h3"
                                        >
                                            Sample Data
                                        </Header>
                                    }
                                >
                                    <KeyValuePairs
                                        columns={1}
                                        items={[
                                            {
                                                label: 'Prefer to Load sample Data',
                                                value: '',
                                            },
                                        ]}
                                    />
                                </Container>
                            </>
                        )}
                        {wizardData?.sampleLoaded === 'manual' && (
                            <>
                                <Container
                                    header={
                                        <Header
                                            variant="h2"
                                            headingTagOverride="h3"
                                        >
                                            AWS Account
                                        </Header>
                                    }
                                >
                                    <KeyValuePairs
                                        columns={3}
                                        items={[
                                            {
                                                label: 'Access Key',
                                                value: wizardData?.awsData
                                                    ?.accessKey,
                                            },
                                            {
                                                label: 'Access Secret',
                                                value: wizardData?.awsData
                                                    ?.accessSecret,
                                            },
                                            {
                                                label: 'Role',
                                                value: wizardData?.awsData
                                                    ?.role,
                                            },
                                        ]}
                                    />
                                </Container>
                                <Container
                                    header={
                                        <Header
                                            variant="h2"
                                            headingTagOverride="h3"
                                        >
                                            Azure Account
                                        </Header>
                                    }
                                >
                                    <KeyValuePairs
                                        columns={4}
                                        items={[
                                            {
                                                label: 'Application (client) ID',
                                                value: wizardData?.azureData
                                                    ?.applicationId,
                                            },
                                            {
                                                label: 'Object  ID',
                                                value: wizardData?.azureData
                                                    ?.objectId,
                                            },
                                            {
                                                label: 'Directory (Tenant) ID',
                                                value: wizardData?.azureData
                                                    ?.directoryId,
                                            },
                                            {
                                                label: 'Secret Value',
                                                value: wizardData?.azureData
                                                    ?.secretValue,
                                            },
                                          
                                        ]}
                                    />
                                </Container>
                            </>
                        )}
                    </SpaceBetween>
                </SpaceBetween>
            </SpaceBetween>
        </Box>
    )
}
