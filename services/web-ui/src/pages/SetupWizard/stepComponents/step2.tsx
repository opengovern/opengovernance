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
import { link } from 'fs'
import { useEffect, useState } from 'react'
import axios from 'axios'
import ReactMarkdown from 'react-markdown'
import { WizarData } from '../types'
import {
    Box,
    Container,
    Header,
    FormField,
    RadioGroup,
    SpaceBetween,
    Select,
    Tiles,
    TextContent,
    Link,
    Input,
} from '@cloudscape-design/components'
interface Props {
    setLoading: Function
    wizardData: WizarData
    setWizardData: Function
}
interface InputDetail {
    detail: {
        value: string
    }
}

export default function UserCredinations({
    setLoading,
    wizardData,
    setWizardData,
}: Props) {
    const workspace = useParams<{ ws: string }>().ws
    const navigate = useNavigate()
    const [checked, setChecked] = useState(false)
    const [markdown, setMarkdown] = useState('')
    const [emailError, setEmailError] = useState('')
    const EmailChecker = (email: string) => {
        if (email === '' && wizardData?.userData) {
            setEmailError('Please enter your email')
        } else {
            if (email.includes('@')) {
                setEmailError('')
            } else {
                setEmailError('Please enter a valid email')
            }
        }
    }
    useEffect(() => {
      
    }, [])
    return (
        <Box margin={{ bottom: 'l' }}>
            <Container header={<Header variant="h2">Setup Login </Header>}>
                <Flex
                    flexDirection="col"
                    justifyContent="start"
                    alignItems="start"
                    className="gap-3 w-full"
                >
                    <FormField
                        // constraintText="Requirements and constraints for the field."
                        className="w-full"
                        // stretch
                        description={''}
                        errorText={emailError}
                        label="Please enter your email and password"
                    >
                        {/* @ts-ignore */}
                        <Input
                            className="w-2/3"
                            placeholder="Email "
                            value={wizardData?.userData?.email || ''}
                            onChange={({ detail }: InputDetail) => {
                                EmailChecker(detail.value)
                                setWizardData({
                                    ...wizardData,
                                    userData: {
                                        ...wizardData?.userData,
                                        email: detail.value,
                                    },
                                })
                            }}
                        />
                        {/* @ts-ignore */}
                    </FormField>
                    <FormField
                        // constraintText="Requirements and constraints for the field."
                        className="w-full"
                        // stretch
                        description={''}
                        
                        errorText={''}
                        label=""
                    >
                        {/* @ts-ignore */}
                        <Input
                            className="w-2/3"
                            placeholder="Password"
                            type='password'
                            value={wizardData?.userData?.password || ''}
                            onChange={({ detail }: InputDetail) =>
                                setWizardData({
                                    ...wizardData,
                                    userData: {
                                        ...wizardData?.userData,
                                        password: detail.value,
                                    },
                                })
                            }
                        />
                        {/* @ts-ignore */}
                    </FormField>
                </Flex>
            </Container>
        </Box>
    )
}
