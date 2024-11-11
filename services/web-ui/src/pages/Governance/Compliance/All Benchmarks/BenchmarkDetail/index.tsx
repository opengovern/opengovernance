import { Link, useParams } from 'react-router-dom'
import { useAtomValue, useSetAtom } from 'jotai'
import {
    Button,
    Card,
    Flex,
    Grid,
    List,
    ListItem,
    Tab,
    TabGroup,
    TabList,
    TabPanel,
    TabPanels,
    Text,
    Title,
} from '@tremor/react'
import { useEffect } from 'react'
import ReactJson from '@microlink/react-json-view'
import { CheckCircleIcon, PlayCircleIcon, XCircleIcon } from '@heroicons/react/24/outline'
import {
    GithubComKaytuIoKaytuEnginePkgBenchmarkApiListV3ResponseMetaData,
    GithubComKaytuIoKaytuEnginePkgComplianceApiConformanceStatus,
    GithubComKaytuIoKaytuEnginePkgComplianceApiResourceFinding,
    GithubComKaytuIoKaytuEnginePkgInventoryApiSmartQueryItem,
    GithubComKaytuIoKaytuEnginePkgInventoryApiSmartQueryItemV2,
} from '../../../../../api/api'
import DrawerPanel from '../../../../../components/DrawerPanel'
import { getConnectorIcon } from '../../../../../components/Cards/ConnectorCard'
import SummaryCard from '../../../../../components/Cards/SummaryCard'
import { useComplianceApiV1FindingsResourceCreate } from '../../../../../api/compliance.gen'
import Spinner from '../../../../../components/Spinner'
// import { severityBadge } from '../Controls'
import { isDemoAtom, notificationAtom, queryAtom } from '../../../../../store'
// import Timeline from '../FindingsWithFailure/Detail/Timeline'
import { searchAtom } from '../../../../../utilities/urlstate'
import { dateTimeDisplay } from '../../../../../utilities/dateDisplay'
import Editor from 'react-simple-code-editor'
import { highlight, languages } from 'prismjs' // eslint-disable-next-line import/no-extraneous-dependencies
import 'prismjs/components/prism-sql' // eslint-disable-next-line import/no-extraneous-dependencies
import 'prismjs/themes/prism.css'

interface IResourceFindingDetail {
    selectedItem:
        | GithubComKaytuIoKaytuEnginePkgBenchmarkApiListV3ResponseMetaData
        | undefined
    open: boolean
    onClose: () => void
    onRefresh: () => void
    linkPrefix?: string
}

export default function BenchmarkDetail({
    selectedItem,
    open,
    onClose,
    onRefresh,
    linkPrefix = '',
}: IResourceFindingDetail) {
    const { ws } = useParams()
    const setQuery = useSetAtom(queryAtom)

    // const { response, isLoading, sendNow } =
    //     useComplianceApiV1FindingsResourceCreate(
    //         { kaytuResourceId: resourceFinding?.kaytuResourceID || '' },
    //         {},
    //         false
    //     )
    const searchParams = useAtomValue(searchAtom)

    // useEffect(() => {
    //     if (resourceFinding && open) {
    //         sendNow()
    //     }
    // }, [resourceFinding, open])

    const isDemo = useAtomValue(isDemoAtom)

    // const finding = resourceFinding?.findings
    //     ?.filter((f) => f.controlID === controlID)
    //     .at(0)

    // const conformance = () => {
    //     if (showOnlyOneControl) {
    //         return (finding?.conformanceStatus || 0) ===
    //             GithubComKaytuIoKaytuEnginePkgComplianceApiConformanceStatus.ConformanceStatusFailed ? (
    //             <Flex className="w-fit gap-1.5">
    //                 <XCircleIcon className="h-4 text-rose-600" />
    //                 <Text>Failed</Text>
    //             </Flex>
    //         ) : (
    //             <Flex className="w-fit gap-1.5">
    //                 <CheckCircleIcon className="h-4 text-emerald-500" />
    //                 <Text>Passed</Text>
    //             </Flex>
    //         )
    //     }

    //     const failingControls = new Map<string, string>()
    //     resourceFinding?.findings?.forEach((f) => {
    //         failingControls.set(f.controlID || '', '')
    //     })

    //     return failingControls.size > 0 ? (
    //         <Flex className="w-fit gap-1.5">
    //             <XCircleIcon className="h-4 text-rose-600" />
    //             <Text>{failingControls.size} Failing</Text>
    //         </Flex>
    //     ) : (
    //         <Flex className="w-fit gap-1.5">
    //             <CheckCircleIcon className="h-4 text-emerald-500" />
    //             <Text>Passed</Text>
    //         </Flex>
    //     )
    // }

    return (
        <DrawerPanel
            open={open}
            onClose={onClose}
            title={
                <Flex justifyContent="start">
                    {selectedItem?.connectors &&
                        getConnectorIcon(selectedItem?.connectors)}
                    <Title className="text-lg font-semibold ml-2 my-1">
                        {selectedItem?.title}
                    </Title>
                </Flex>
            }
        >
            <TabGroup>
                <TabList>
                    <Tab>Summary</Tab>
                    <Tab>Services</Tab>
                </TabList>

                <TabPanels>
                    <TabPanel>
                        <Grid className="w-full gap-4 mb-6" numItems={1}>
                            <Flex
                                flexDirection="row"
                                justifyContent="between"
                                alignItems="start"
                                className="mt-2"
                            >
                                <Text className="w-56 font-bold">ID : </Text>
                                <Text className="w-full">
                                    {selectedItem?.id}
                                </Text>
                            </Flex>
                            <Flex
                                flexDirection="row"
                                justifyContent="between"
                                alignItems="start"
                                className="mt-2"
                            >
                                <Text className="w-56 font-bold">Title : </Text>
                                <Text className="w-full">
                                    {selectedItem?.title}
                                </Text>
                            </Flex>{' '}
                            <Flex
                                flexDirection="row"
                                justifyContent="between"
                                alignItems="start"
                                className="mt-2"
                            >
                                <Text className="w-56 font-bold">
                                    Description :{' '}
                                </Text>
                                <Text className="w-full">
                                    {selectedItem?.description}
                                </Text>
                            </Flex>{' '}
                            <Flex
                                flexDirection="row"
                                justifyContent="between"
                                alignItems="start"
                                className="mt-2"
                            >
                                <Text className="w-56 font-bold">
                                    Connector :{' '}
                                </Text>
                                <Text className="w-full">
                                    {selectedItem?.connectors?.map(
                                        (item, index) => {
                                            return `${item} `
                                        }
                                    )}
                                </Text>
                            </Flex>
                            <Flex
                                flexDirection="row"
                                justifyContent="between"
                                alignItems="start"
                                className="mt-2"
                            >
                                <Text className="w-56 font-bold">
                                    Created At :{' '}
                                </Text>
                                <Text className="w-full">
                                    {`${
                                        selectedItem?.created_at.split('T')[0]
                                    } ${
                                        selectedItem?.created_at.split('T')[1].split(".")[0]
                                    }`}
                                </Text>
                            </Flex>
                        </Grid>
                    </TabPanel>
                    <TabPanel>
                        {/* 
                        table for primary tables
                        */}
                        <Grid className="w-full gap-4 mb-6" numItems={1}>
                            <Flex
                                flexDirection="row"
                                justifyContent="between"
                                alignItems="start"
                                className="mt-2"
                            >
                                <Text className="w-56 font-bold">Primary Tables : </Text>
                                <Text className="w-full">
                                    {selectedItem?.primary_tables?.map(
                                        (item, index) => {
                                            return `${item} `
                                        }
                                    )}
                                </Text>
                            </Flex>
                        </Grid>
                    </TabPanel>
                </TabPanels>
            </TabGroup>
        </DrawerPanel>
    )
}
