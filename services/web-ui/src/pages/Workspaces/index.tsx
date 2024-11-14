import { Button, Card, Flex, Grid, Text, Title } from '@tremor/react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { ArrowPathIcon } from '@heroicons/react/24/outline'
import { useEffect } from 'react'
import { useWorkspaceApiV1WorkspacesList } from '../../api/workspace.gen'
import WorkspaceCard from '../../components/Cards/WorkspaceCard'
import Spinner from '../../components/Spinner'
import TopHeader from '../../components/Layout/Header'
import { OpenGovernance } from '../../icons/icons'
import Controls from '../Governance/Controls'

export default function Workspaces() {
    const navigate = useNavigate()
    const [searchParams, setSearchParams] = useSearchParams()

    const {
        response: workspaces,
        isLoading,
        isExecuted,
        sendNow: refreshList,
    } = useWorkspaceApiV1WorkspacesList()

    useEffect(() => {
        if (isExecuted && !isLoading) {
            if (workspaces?.length === 1 && searchParams.has('onLogin')) {
                window.location.href = `/ws/${workspaces.at(0)?.name}`
            }
        }
    }, [isLoading])

    // if (workspaces?.length === 0) {
    //     return (
    //         <Flex flexDirection="col">
    //             <Card className="w-1/2 pt-12 pb-16 mt-40">
    //                 <Flex
    //                     flexDirection="col"
    //                     justifyContent="start"
    //                     alignItems="center"
    //                 >
    //                     <OpenGovernance className="w-14 h-14 mb-6" />
    //                     <Title className="font-bold text-2xl mb-3">
    //                         Welcome
    //                     </Title>
    //                     <Text>
    //                         Your account doesn’t have access to any
    //                         organizations or workspaces.
    //                     </Text>
    //                     <Text className="mb-6">
    //                         If you wish to try our platform, please{' '}
    //                         <b>Request a no-cost trial access.</b>
    //                     </Text>
    //                     <a
    //                         href="https://kaytu.io/bookademo/"
    //                         target="_blank"
    //                         rel="noreferrer"
    //                     >
    //                         <Button variant="secondary">
    //                             Request Free Trial
    //                         </Button>
    //                     </a>
    //                 </Flex>
    //             </Card>
    //             <Text className="mt-8 text-gray-400">
    //                 If you think this is an error or you’ve lost access to an
    //                 Organization,
    //             </Text>
    //             <Text className="text-gray-400">
    //                 please send an email to{' '}
    //                 <b className="text-gray-600">support@kaytu.io</b>
    //             </Text>
    //         </Flex>
    //     )
    // }

    return (
        <>
            <TopHeader>
                {!isLoading && (
                    <Flex className="w-fit gap-3">
                        <Button variant="secondary" onClick={refreshList}>
                            <ArrowPathIcon className="h-5 text-openg-500" />
                        </Button>
                        {/* <Button
                            variant="secondary"
                            onClick={() => navigate(`/ws/new-ws`)}
                        >
                            Add new OpenGovernance workspace
                        </Button> */}
                    </Flex>
                )}
            </TopHeader>
            {isLoading ? (
                <Flex justifyContent="center" className="mt-56">
                    <Spinner />
                </Flex>
            ) : (
                <Flex justifyContent="center" flexDirection="row">
                    <div className="max-w-6xl w-2/3">
                        <Grid numItems={1} className="gap-4">
                            {workspaces?.map((ws) => {
                                return (
                                    <WorkspaceCard
                                        workspace={ws}
                                        refreshList={refreshList}
                                    />
                                )
                            })}
                        </Grid>
                    </div>
                </Flex>
            )}
        </>
    )
}
