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
import Spinner from '../../../components/Spinner'
interface Props {
    setLoading: Function
}


export default function License({ setLoading }: Props) {
    const workspace = useParams<{ ws: string }>().ws
    const navigate = useNavigate()
    const [checked, setChecked] = useState(false)
    const [loadingMD, setLoadingMD] = useState(false)

        const [markdown, setMarkdown] = useState('')
    useEffect(() => {
        setLoadingMD(true)
        const config = {
            headers: {
                'Content-Type': 'text/plain',
            },
            withCredentials: false,
        }
        axios
            .get(
                'https://raw.githubusercontent.com/kaytu-io/open-governance/refs/heads/main/LICENSE.md',
                config
            )
            .then((res) => {
                setMarkdown(res.data)
        setLoadingMD(false)

            })
    }, [])
    return (
        <>
            <Card>
                {loadingMD ? (
                    <>
                        <Spinner />
                    </>
                ) : (
                    <>
                        <Flex
                            className="gap-2"
                            flexDirection="col"
                            justifyContent="start"
                            alignItems="start"
                        >
                            <ReactMarkdown
                                children={markdown}
                                // linkTarget="_blank"
                                // transformLinkUri={undefined}
                            />
                        </Flex>
                    </>
                )}
            </Card>
            <Checkbox className='mt-3'
                onChange={({ detail }) => {
                    setChecked(detail.checked)
                    setLoading(!detail.checked)
                }}
                checked={checked}
            >
                I Acknowledge That I Have Read And Understand The Terms
            </Checkbox>
        </>
    )
}
