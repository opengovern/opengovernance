import { useNavigate, useParams, useSearchParams } from 'react-router-dom'
import {
    Badge,
    BarList,
    Button,
    Callout,
    Card,
    Col,
    Flex,
    Grid,
    List,
    ListItem,
    Select,
    Tab,
    TabGroup,
    TabList,
    TabPanel,
    TabPanels,
    Text,
    Title,
} from '@tremor/react'
import { useAtomValue } from 'jotai'
import { useEffect, useMemo, useState } from 'react'
import {
    ChevronDownIcon,
    ChevronRightIcon,
    ChevronUpIcon,
} from '@heroicons/react/24/outline'
import { ICellRendererParams } from 'ag-grid-community'
import {
    useInventoryApiV2AnalyticsTrendList,
    useInventoryApiV2ResourceCollectionDetail,
    useInventoryApiV2ResourceCollectionLandscapeDetail,
} from '../../../api/inventory.gen'
import Spinner from '../../../components/Spinner'
import {
    useComplianceApiV1AssignmentsResourceCollectionDetail,
    useComplianceApiV1BenchmarksSummaryList,
} from '../../../api/compliance.gen'
import {
    GithubComKaytuIoKaytuEnginePkgComplianceApiAssignedBenchmark,
    GithubComKaytuIoKaytuEnginePkgComplianceApiBenchmarkEvaluationSummary,
    GithubComKaytuIoKaytuEngineServicesIntegrationApiEntityListConnectionsSummaryResponse,
} from '../../../api/api'
import Chart from '../../../components/Chart'
import {
    camelCaseToLabel,
    capitalizeFirstLetter,
} from '../../../utilities/labelMaker'
import { dateDisplay, dateTimeDisplay } from '../../../utilities/dateDisplay'
import Table, { IColumn } from '../../../components/Table'
import {
    checkGranularity,
    generateItems,
} from '../../../utilities/dateComparator'
import SummaryCard from '../../../components/Cards/SummaryCard'
import { numberDisplay } from '../../../utilities/numericDisplay'
import { BarChartIcon, LineChartIcon } from '../../../icons/icons'
import { generateVisualMap, resourceTrendChart } from '../../Assets'
import { useIntegrationApiV1ConnectionsSummariesList } from '../../../api/integration.gen'
import Landscape from '../../../components/Landscape'
import Tag from '../../../components/Tag'
import DrawerPanel from '../../../components/DrawerPanel'
import { getConnectorIcon } from '../../../components/Cards/ConnectorCard'
import { benchmarkChecks } from '../../../components/Cards/ComplianceCard'
import TopHeader from '../../../components/Layout/Header'
import { options } from '../../Assets/Metric/Table'
import {
    defaultTime,
    searchAtom,
    useUrlDateRangeState,
} from '../../../utilities/urlstate'

// const pieData = (
//     input:
//         | GithubComKaytuIoKaytuEnginePkgComplianceApiGetBenchmarksSummaryResponse
//         | undefined
// ) => {
//     const data: any[] = []
//     if (input && input.totalChecks) {
//         // eslint-disable-next-line array-callback-return
//         Object.entries(input.totalChecks).map(([key, value]) => {
//             data.push({ name: camelCaseToLabel(key), value })
//         })
//     }
//     return data.reverse()
// }

const barData = (
    input:
        | GithubComKaytuIoKaytuEngineServicesIntegrationApiEntityListConnectionsSummaryResponse
        | undefined
) => {
    const data: any[] = []
    if (input && input.connections) {
        // eslint-disable-next-line array-callback-return
        input.connections.map((c, i) => {
            if (i < 5) {
                data.push({
                    name: c.providerConnectionName,
                    value: c.resourceCount || 0,
                })
            }
        })
    }
    return data
}

const complianceColumns: IColumn<any, any>[] = [
    {
        width: 140,
        field: 'connectors',
        headerName: 'Cloud provider',
        sortable: true,
        filter: true,
        enableRowGroup: true,
        type: 'string',
    },
    {
        field: 'title',
        headerName: 'Benchmark title',
        sortable: true,
        filter: true,
        enableRowGroup: true,
        type: 'string',
        cellRenderer: (
            param: ICellRendererParams<
                | GithubComKaytuIoKaytuEnginePkgComplianceApiBenchmarkEvaluationSummary
                | undefined
            >
        ) =>
            param.value && (
                <Flex flexDirection="col" alignItems="start">
                    <Text>{param.value}</Text>
                    <Flex justifyContent="start" className="mt-1 gap-2">
                        {param.data?.tags?.category?.map((cat) => (
                            <Badge color="slate" size="xs">
                                {cat}
                            </Badge>
                        ))}
                        {param.data?.tags?.kaytu_category?.map((cat) => (
                            <Badge color="emerald" size="xs">
                                {cat}
                            </Badge>
                        ))}
                        {!!param.data?.tags?.cis && (
                            <Badge color="sky" size="xs">
                                CIS
                            </Badge>
                        )}
                        {!!param.data?.tags?.hipaa && (
                            <Badge color="blue" size="xs">
                                Hipaa
                            </Badge>
                        )}
                    </Flex>
                </Flex>
            ),
    },
    {
        headerName: 'Security score',
        width: 150,
        sortable: true,
        filter: true,
        enableRowGroup: true,
        type: 'number',
        cellRenderer: (
            param: ICellRendererParams<
                | GithubComKaytuIoKaytuEnginePkgComplianceApiBenchmarkEvaluationSummary
                | undefined
            >
        ) =>
            param.data &&
            `${(
                ((param.data?.conformanceStatusSummary?.passed || 0) /
                    (benchmarkChecks(param.data).total || 1)) *
                100
            ).toFixed(2)} %`,
    },
    {
        width: 120,
        field: 'status',
        headerName: 'Status',
        sortable: true,
        filter: true,
        enableRowGroup: true,
        rowGroup: true,
        type: 'string',
    },
]

// const bmList = (
//     assignmentList:
//         | GithubComKaytuIoKaytuEnginePkgComplianceApiAssignedBenchmark[]
//         | undefined,
//     summaryList:
//         | GithubComKaytuIoKaytuEnginePkgComplianceApiGetBenchmarksSummaryResponse
//         | undefined
// ) => {
//     const rows = []
//     if (assignmentList && summaryList) {
//         for (let i = 0; i < assignmentList.length; i += 1) {
//             const benchmark = summaryList.benchmarkSummary?.find(
//                 (bm) => bm.id === assignmentList[i].benchmarkId?.id
//             )
//             if (assignmentList[i].status) {
//                 rows.push({
//                     ...assignmentList[i].benchmarkId,
//                     ...benchmark,
//                     status: 'Assigned',
//                 })
//             } else {
//                 rows.push({
//                     ...assignmentList[i].benchmarkId,
//                     ...benchmark,
//                     status: 'Not assigned',
//                 })
//             }
//         }
//     }
//     return rows
// }

export default function ResourceCollectionDetail() {
    const { ws } = useParams()
    const { resourceId } = useParams()
    const { value: activeTimeRange } = useUrlDateRangeState(
        defaultTime(ws || '')
    )
    const navigate = useNavigate()
    const searchParams = useAtomValue(searchAtom)
    const [openDrawer, setOpenDrawer] = useState(false)
    const [showSummary, setShowSummary] = useState(false)

    const [selectedChart, setSelectedChart] = useState<'line' | 'bar'>('line')
    const [selectedIndex, setSelectedIndex] = useState(0)
    const [selectedGranularity, setSelectedGranularity] = useState<
        'monthly' | 'daily' | 'yearly'
    >(
        checkGranularity(activeTimeRange.start, activeTimeRange.end).daily
            ? 'daily'
            : 'monthly'
    )
    useEffect(() => {
        setSelectedGranularity(
            checkGranularity(activeTimeRange.start, activeTimeRange.end).monthly
                ? 'monthly'
                : 'daily'
        )
    }, [activeTimeRange])
    const [selectedDatapoint, setSelectedDatapoint] = useState<any>(undefined)

    useEffect(() => {
        if (selectedIndex === 0) setSelectedChart('line')
        if (selectedIndex === 1) setSelectedChart('bar')
    }, [selectedIndex])

    const query = {
        resourceCollection: [resourceId || ''],
        startTime: activeTimeRange.start.unix(),
        endTime: activeTimeRange.end.unix(),
    }

    const { response: detail, isLoading: detailsLoading } =
        useInventoryApiV2ResourceCollectionDetail(resourceId || '')
    const { response: complianceKPI, isLoading: complianceKPILoading } =
        useComplianceApiV1BenchmarksSummaryList({
            resourceCollection: [resourceId || ''],
        })
    const { response } = useComplianceApiV1AssignmentsResourceCollectionDetail(
        resourceId || ''
    )
    const { response: accountInfo, isLoading: accountInfoLoading } =
        useIntegrationApiV1ConnectionsSummariesList({
            // eslint-disable-next-line @typescript-eslint/ban-ts-comment
            // @ts-ignore
            ...query,
            needCost: false,
            sortBy: 'resource_count',
        })
    const { response: resourceTrend, isLoading: resourceTrendLoading } =
        useInventoryApiV2AnalyticsTrendList({
            ...query,
            // eslint-disable-next-line @typescript-eslint/ban-ts-comment
            // @ts-ignore
            granularity: selectedGranularity,
        })
    const { response: landscape, isLoading: landscapeLoading } =
        useInventoryApiV2ResourceCollectionLandscapeDetail(resourceId || '')

    // const rows = useMemo(() => bmList(response, complianceKPI), [response])

    return (
        <>
            <TopHeader
                breadCrumb={[
                    detail ? detail.name : 'Resource collection detail',
                ]}
                supportedFilters={['Date']}
                initialFilters={['Date']}
            />
            <Flex alignItems="end" className="mb-4">
                <Flex flexDirection="col" alignItems="start">
                    <Title className="font-semibold">{detail?.name}</Title>
                    <Text>{detail?.description}</Text>
                </Flex>
                <Button
                    variant="light"
                    onClick={() => setShowSummary(!showSummary)}
                    icon={showSummary ? ChevronUpIcon : ChevronDownIcon}
                >{`${showSummary ? 'Hide' : 'Show'} summary`}</Button>
            </Flex>
            {showSummary && (
                <Grid numItems={2} className="w-full gap-4 mb-4">
                    <Card>
                        <Flex
                            flexDirection="col"
                            alignItems="start"
                            className="h-full"
                        >
                            {detailsLoading ? (
                                <Spinner className="my-24" />
                            ) : (
                                <List>
                                    <ListItem>
                                        <Text>Connector</Text>
                                        <Text className="text-gray-800">
                                            {getConnectorIcon(
                                                detail?.connectors
                                            )}
                                        </Text>
                                    </ListItem>
                                    <ListItem>
                                        <Text>Resources</Text>
                                        <Text className="text-gray-800">
                                            {numberDisplay(
                                                detail?.resource_count,
                                                0
                                            )}
                                        </Text>
                                    </ListItem>
                                    <ListItem>
                                        <Text>Services used</Text>
                                        <Text className="text-gray-800">
                                            {detail?.metric_count}
                                        </Text>
                                    </ListItem>
                                    <ListItem>
                                        <Text>Status</Text>
                                        <Text className="text-gray-800">
                                            {capitalizeFirstLetter(
                                                detail?.status || ''
                                            )}
                                        </Text>
                                    </ListItem>
                                    <ListItem>
                                        <Text>Last evaluation</Text>
                                        <Text className="text-gray-800">
                                            {dateTimeDisplay(
                                                detail?.last_evaluated_at
                                            )}
                                        </Text>
                                    </ListItem>
                                    <ListItem>
                                        <Text>Tags</Text>
                                        <Flex
                                            justifyContent="end"
                                            className="w-2/3 flex-wrap gap-1"
                                        >
                                            {/* eslint-disable-next-line @typescript-eslint/ban-ts-comment */}
                                            {/* @ts-ignore */}
                                            {Object.entries(detail?.tags).map(
                                                ([name, value]) => (
                                                    <Tag
                                                        text={`${name}: ${value}`}
                                                    />
                                                )
                                            )}
                                        </Flex>
                                    </ListItem>
                                </List>
                            )}
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
                                title="Resource collection detail"
                                open={openDrawer}
                                onClose={() => setOpenDrawer(false)}
                            >
                                <List>
                                    <ListItem>
                                        <Text>Connector</Text>
                                        <Text className="text-gray-800">
                                            {getConnectorIcon(
                                                detail?.connectors
                                            )}
                                        </Text>
                                    </ListItem>
                                    <ListItem className="py-6">
                                        <Text>Resources</Text>
                                        <Text className="text-gray-800">
                                            {numberDisplay(
                                                detail?.resource_count,
                                                0
                                            )}
                                        </Text>
                                    </ListItem>
                                    <ListItem className="py-6">
                                        <Text>Services used</Text>
                                        <Text className="text-gray-800">
                                            {detail?.metric_count}
                                        </Text>
                                    </ListItem>
                                    <ListItem className="py-6">
                                        <Text>Status</Text>
                                        <Text className="text-gray-800">
                                            {capitalizeFirstLetter(
                                                detail?.status || ''
                                            )}
                                        </Text>
                                    </ListItem>
                                    <ListItem className="py-6">
                                        <Text>Date created</Text>
                                        <Text className="text-gray-800">
                                            {dateTimeDisplay(
                                                detail?.created_at
                                            )}
                                        </Text>
                                    </ListItem>
                                    <ListItem className="py-6">
                                        <Text>Last evaluation</Text>
                                        <Text className="text-gray-800">
                                            {dateTimeDisplay(
                                                detail?.last_evaluated_at
                                            )}
                                        </Text>
                                    </ListItem>
                                    <ListItem className="py-6">
                                        <Text>Tags</Text>
                                        <Flex
                                            justifyContent="end"
                                            className="w-2/3 flex-wrap gap-1"
                                        >
                                            {/* eslint-disable-next-line @typescript-eslint/ban-ts-comment */}
                                            {/* @ts-ignore */}
                                            {Object.entries(detail?.tags).map(
                                                ([name, value]) => (
                                                    <Tag
                                                        text={`${name}: ${value}`}
                                                    />
                                                )
                                            )}
                                        </Flex>
                                    </ListItem>
                                </List>
                            </DrawerPanel>
                        </Flex>
                    </Card>
                    <Card className="h-full">
                        <TabGroup>
                            <Flex>
                                <Title className="font-semibold">KPI</Title>
                                <TabList className="w-1/2">
                                    <Tab>Compliance</Tab>
                                    <Tab>Infrastructure</Tab>
                                </TabList>
                            </Flex>
                            <TabPanels>
                                <TabPanel className="mt-0">
                                    {/* <Chart
                                        labels={[]}
                                        chartType="doughnut"
                                        chartAggregation="trend"
                                        chartData={pieData(complianceKPI)}
                                        loading={complianceKPILoading}
                                        colorful
                                    /> */}
                                </TabPanel>
                                <TabPanel>
                                    <Title className="font-semibold mb-3">
                                        Top accounts
                                    </Title>
                                    <BarList
                                        data={barData(accountInfo)}
                                        color="slate"
                                    />
                                </TabPanel>
                            </TabPanels>
                        </TabGroup>
                    </Card>
                </Grid>
            )}
            <TabGroup>
                <TabList className="mb-3">
                    <Tab>Landscape</Tab>
                    <Tab>Compliance</Tab>
                    <Tab>Infrastructure</Tab>
                </TabList>
                <TabPanels>
                    <TabPanel>
                        <Landscape
                            data={landscape}
                            isLoading={landscapeLoading}
                        />
                    </TabPanel>
                    <TabPanel>
                        {/* <Table
                            title={`${detail?.name} benchmarks`}
                            downloadable
                            id="resource_collection_bm"
                            rowData={rows}
                            columns={complianceColumns}
                            onRowClicked={(event) => {
                                if (event.data) {
                                    if (event.data.status === 'Assigned') {
                                        navigate(
                                            `${event.data.id}?${searchParams}`
                                        )
                                    } else {
                                        navigate(
                                            `${event.data.id}/details#assignments?${searchParams}`
                                        )
                                    }
                                }
                            }}
                            options={options}
                        /> */}
                    </TabPanel>
                    <TabPanel>
                        <Card>
                            <Grid numItems={6} className="gap-4">
                                <Col numColSpan={1}>
                                    <SummaryCard
                                        title="Resources"
                                        metric={accountInfo?.totalResourceCount}
                                        url="infrastructure-details#cloud-accounts"
                                        loading={accountInfoLoading}
                                        border={false}
                                    />
                                </Col>
                                <Col numColSpan={3} />
                                <Col numColSpan={2}>
                                    <Flex
                                        justifyContent="end"
                                        className="gap-4"
                                    >
                                        {generateItems(
                                            activeTimeRange.start,
                                            activeTimeRange.end,
                                            capitalizeFirstLetter(
                                                selectedGranularity
                                            ),
                                            selectedGranularity,
                                            (v) => {
                                                // eslint-disable-next-line @typescript-eslint/ban-ts-comment
                                                // @ts-ignore
                                                setSelectedGranularity(v)
                                            }
                                        )}
                                        <TabGroup
                                            index={selectedIndex}
                                            onIndexChange={setSelectedIndex}
                                            className="w-fit rounded-lg"
                                        >
                                            <TabList variant="solid">
                                                <Tab value="line">
                                                    <LineChartIcon className="h-5" />
                                                </Tab>
                                                <Tab value="bar">
                                                    <BarChartIcon className="h-5" />
                                                </Tab>
                                            </TabList>
                                        </TabGroup>
                                    </Flex>
                                </Col>
                            </Grid>
                            {resourceTrend
                                ?.filter(
                                    (t) =>
                                        selectedDatapoint?.color ===
                                            '#E01D48' &&
                                        dateDisplay(t.date) ===
                                            selectedDatapoint?.name
                                )
                                .map((t) => (
                                    <Callout
                                        color="rose"
                                        title="Incomplete data"
                                        className="w-fit mt-4"
                                    >
                                        Checked{' '}
                                        {numberDisplay(
                                            t.totalSuccessfulDescribedConnectionCount,
                                            0
                                        )}{' '}
                                        accounts out of{' '}
                                        {numberDisplay(
                                            t.totalConnectionCount,
                                            0
                                        )}{' '}
                                        on {dateDisplay(t.date)}
                                    </Callout>
                                ))}
                            <Flex justifyContent="end" className="mt-2 gap-2.5">
                                <div className="h-2.5 w-2.5 rounded-full bg-openg-800" />
                                <Text>Resources</Text>
                            </Flex>
                            <Chart
                                labels={
                                    resourceTrendChart(
                                        resourceTrend,
                                        selectedGranularity
                                    ).label
                                }
                                chartData={
                                    resourceTrendChart(
                                        resourceTrend,
                                        selectedGranularity
                                    ).data
                                }
                                chartType={selectedChart}
                                chartAggregation="trend"
                                loading={resourceTrendLoading}
                                visualMap={
                                    generateVisualMap(
                                        resourceTrendChart(
                                            resourceTrend,
                                            selectedGranularity
                                        ).flag,
                                        resourceTrendChart(
                                            resourceTrend,
                                            selectedGranularity
                                        ).label
                                    ).visualMap
                                }
                                markArea={
                                    generateVisualMap(
                                        resourceTrendChart(
                                            resourceTrend,
                                            selectedGranularity
                                        ).flag,
                                        resourceTrendChart(
                                            resourceTrend,
                                            selectedGranularity
                                        ).label
                                    ).markArea
                                }
                                onClick={(p) => setSelectedDatapoint(p)}
                            />
                        </Card>
                    </TabPanel>
                </TabPanels>
            </TabGroup>
        </>
    )
}
