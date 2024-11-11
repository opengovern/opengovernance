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
import { useParams } from 'react-router-dom'
import clipboardCopy from 'clipboard-copy'
import { ChevronRightIcon, Square2StackIcon } from '@heroicons/react/24/outline'
import { useEffect, useState } from 'react'
import { useAtomValue, useSetAtom } from 'jotai'
import {
    IServerSideDatasource,
    RowClickedEvent,
    SortModelItem,
    IServerSideGetRowsParams,
} from 'ag-grid-community'
import { useIntegrationApiV1ConnectionsSummariesList } from '../../../../../api/integration.gen'
import Spinner from '../../../../../components/Spinner'
import { dateTimeDisplay } from '../../../../../utilities/dateDisplay'
import DrawerPanel from '../../../../../components/DrawerPanel'
import { RenderObject } from '../../../../../components/RenderObject'
import { isDemoAtom, notificationAtom } from '../../../../../store'
import {
    useComplianceApiV1AssignmentsConnectionDetail,
    useComplianceApiV1BenchmarksSummaryDetail,
    useComplianceApiV1FindingsCreate,
} from '../../../../../api/compliance.gen'
import Table from '../../../../../components/Table'
import { columns } from '../../../Findings/FindingsWithFailure'
import Breakdown from '../../../../../components/Breakdown'
import FindingDetail from '../../../Findings/FindingsWithFailure/Detail'
import { benchmarkChecks } from '../../../../../components/Cards/ComplianceCard'
import { policyColumns } from '../TopDetails/Controls'
import TopHeader from '../../../../../components/Layout/Header'

export default function SingleComplianceConnection() {
    const [openDrawer, setOpenDrawer] = useState(false)
    const { connectionId, resourceId } = useParams()
    const setNotification = useSetAtom(notificationAtom)
    const isDemo = useAtomValue(isDemoAtom)
    const [sortModel, setSortModel] = useState<SortModelItem[]>([])
    const [openFinding, setOpenFinding] = useState(false)
    const [finding, setFinding] = useState<any>(undefined)

    const query = {
        ...(connectionId && {
            connectionId: [connectionId.replace('account_', '')],
        }),
        ...(resourceId && {
            resourceCollection: [resourceId],
        }),
    }
    const { response: accountInfo, isLoading: accountInfoLoading } =
        useIntegrationApiV1ConnectionsSummariesList({
            ...query,
            pageSize: 1,
            needCost: false,
        })
    const con = accountInfo?.connections?.at(0)

    const { response: benchmarkList } =
        useComplianceApiV1AssignmentsConnectionDetail(
            connectionId?.replace('account_', '') || ''
        )
    const [benchmark, setBenchmark] = useState(
        benchmarkList?.filter((bm) => bm.status)[0].benchmarkId?.id
    )
    const {
        response: benchmarkDetail,
        isLoading: detailLoading,
        sendNow: updateDetail,
    } = useComplianceApiV1BenchmarksSummaryDetail(
        benchmark || '',
        {
            // eslint-disable-next-line @typescript-eslint/ban-ts-comment
            // @ts-ignore
            connectionID: [connection?.replace('account_', '') || ''],
        },
        {},
        false
    )

    const {
        response: findings,
        isLoading,
        sendNowWithParams: updateFindings,
    } = useComplianceApiV1FindingsCreate({
        filters: {
            // eslint-disable-next-line @typescript-eslint/ban-ts-comment
            // @ts-ignore
            connectionID: [connection?.replace('account_', '') || ''],
        },
        sort: sortModel.length
            ? [{ [sortModel[0].colId]: sortModel[0].sort }]
            : [],
    })

    useEffect(() => {
        if (benchmark) {
            updateDetail()
        }
    }, [benchmark])

    const getData = (sort: SortModelItem[]) => {
        setSortModel(sort)
        updateFindings(
            {
                filters: {
                    // eslint-disable-next-line @typescript-eslint/ban-ts-comment
                    // @ts-ignore
                    connectionID: [connection?.replace('account_', '') || ''],
                },
                sort: sort.length ? [{ [sort[0].colId]: sort[0].sort }] : [],
            },
            {}
        )
    }

    const datasource: IServerSideDatasource = {
        getRows: (params: IServerSideGetRowsParams) => {
            if (params.request.sortModel.length > 0) {
                if (sortModel.length > 0) {
                    if (
                        params.request.sortModel[0].colId !==
                            sortModel[0].colId ||
                        params.request.sortModel[0].sort !== sortModel[0].sort
                    ) {
                        getData([params.request.sortModel[0]])
                    }
                } else {
                    getData([params.request.sortModel[0]])
                }
            } else if (sortModel.length > 0) {
                getData([])
            }
            if (findings) {
                params.success({
                    rowData: findings?.findings || [],
                    rowCount: findings?.totalCount || 0,
                })
            } else {
                params.fail()
            }
        },
    }

    return (
        <>
            <TopHeader
                breadCrumb={[
                    con ? con?.providerConnectionName : 'Single account detail',
                ]}
            />
            <Grid numItems={2} className="w-full gap-4">
                <Card className="w-full">
                    <Flex
                        flexDirection="col"
                        alignItems="start"
                        className="h-full"
                    >
                        <Flex flexDirection="col" alignItems="start">
                            <Title className="font-semibold">
                                Connection details
                            </Title>
                            {accountInfoLoading ? (
                                <Spinner className="my-28" />
                            ) : (
                                <List className="mt-2">
                                    <ListItem>
                                        <Text>Cloud Provider</Text>
                                        <Text className="text-gray-800">
                                            {con?.connector}
                                        </Text>
                                    </ListItem>
                                    <ListItem>
                                        <Text>Discovered name</Text>
                                        <Flex className="gap-1 w-fit">
                                            <Button
                                                variant="light"
                                                onClick={() =>
                                                    clipboardCopy(
                                                        `Discovered name: ${con?.providerConnectionName}`
                                                    ).then(() =>
                                                        setNotification({
                                                            text: 'Discovered name copied to clipboard',
                                                            type: 'info',
                                                        })
                                                    )
                                                }
                                                icon={Square2StackIcon}
                                            />
                                            <Text className="text-gray-800">
                                                {con?.providerConnectionName}
                                            </Text>
                                        </Flex>
                                    </ListItem>
                                    <ListItem>
                                        <Text>Discovered ID</Text>
                                        <Flex className="gap-1 w-fit">
                                            <Button
                                                variant="light"
                                                onClick={() =>
                                                    clipboardCopy(
                                                        `Discovered ID: ${con?.providerConnectionID}`
                                                    ).then(() =>
                                                        setNotification({
                                                            text: 'Discovered ID copied to clipboard',
                                                            type: 'info',
                                                        })
                                                    )
                                                }
                                                icon={Square2StackIcon}
                                            />
                                            <Text className="text-gray-800">
                                                {con?.providerConnectionID}
                                            </Text>
                                        </Flex>
                                    </ListItem>
                                    <ListItem>
                                        <Text>Lifecycle state</Text>
                                        <Text className="text-gray-800">
                                            {con?.lifecycleState}
                                        </Text>
                                    </ListItem>
                                    <ListItem>
                                        <Text>Onboard date</Text>
                                        <Text className="text-gray-800">
                                            {dateTimeDisplay(con?.onboardDate)}
                                        </Text>
                                    </ListItem>
                                    <ListItem>
                                        <Text>Last inventory</Text>
                                        <Text className="text-gray-800">
                                            {dateTimeDisplay(
                                                con?.lastInventory
                                            )}
                                        </Text>
                                    </ListItem>
                                </List>
                            )}
                        </Flex>
                        <Flex justifyContent="end">
                            <Button
                                variant="light"
                                icon={ChevronRightIcon}
                                iconPosition="right"
                                onClick={() => setOpenDrawer(true)}
                            >
                                See more
                            </Button>
                        </Flex>
                        <DrawerPanel
                            title="Connection details"
                            open={openDrawer}
                            onClose={() => setOpenDrawer(false)}
                        >
                            <RenderObject obj={con} />
                        </DrawerPanel>
                    </Flex>
                </Card>
                <Breakdown
                    title={`Severity breakdown for ${benchmark}`}
                    chartData={[
                        {
                            name: 'Critical',
                            value: benchmarkChecks(benchmarkDetail).critical,
                        },
                        {
                            name: 'High',
                            value: benchmarkChecks(benchmarkDetail).high,
                        },
                        {
                            name: 'Medium',
                            value: benchmarkChecks(benchmarkDetail).medium,
                        },
                        {
                            name: 'Low',
                            value: benchmarkChecks(benchmarkDetail).low,
                        },
                        // { name: 'Passed', value: passed },
                        {
                            name: 'None',
                            value: benchmarkChecks(benchmarkDetail).none,
                        },
                    ]}
                    loading={detailLoading}
                />
            </Grid>
            <TabGroup className="mt-4">
                <TabList className="mb-3">
                    {/* eslint-disable-next-line react/jsx-no-useless-fragment */}
                    <>
                        {benchmarkList
                            ?.filter((bm) => bm.status)
                            ?.map((bm) => (
                                <Tab
                                    onClick={() =>
                                        setBenchmark(bm.benchmarkId?.id || '')
                                    }
                                >
                                    {bm.benchmarkId?.title}
                                </Tab>
                            ))}
                        <Tab disabled>Findings</Tab>
                    </>
                </TabList>
                <TabPanels>
                    {benchmarkList
                        ?.filter((bm) => bm.status)
                        ?.map((bm) => (
                            <TabPanel>
                                <Table
                                    title={`${bm.benchmarkId?.title} controls`}
                                    downloadable
                                    id="compliance_policies"
                                    loading={detailLoading}
                                    onGridReady={(e) => {
                                        if (detailLoading) {
                                            e.api.showLoadingOverlay()
                                        }
                                    }}
                                    columns={policyColumns}
                                    // rowData={policies}
                                />
                            </TabPanel>
                        ))}
                    <TabPanel>
                        <Table
                            title="Findings"
                            downloadable
                            id="compliance_findings"
                            columns={columns(isDemo)}
                            onRowClicked={(event: RowClickedEvent) => {
                                setFinding(event.data)
                                setOpenFinding(true)
                            }}
                            onGridReady={(e) => {
                                if (isLoading) {
                                    e.api.showLoadingOverlay()
                                }
                            }}
                            serverSideDatasource={datasource}
                            loading={isLoading}
                        />
                    </TabPanel>
                </TabPanels>
            </TabGroup>
            <FindingDetail
                type="finding"
                finding={finding}
                open={openFinding}
                onClose={() => setOpenFinding(false)}
                onRefresh={() => window.location.reload()}
            />
        </>
    )
}
