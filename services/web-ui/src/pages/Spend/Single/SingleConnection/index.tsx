import {
    Button,
    Callout,
    Card,
    Col,
    Flex,
    Grid,
    List,
    ListItem,
    Select,
    SelectItem,
    Tab,
    TabGroup,
    TabList,
    TabPanel,
    TabPanels,
    Text,
    Title,
} from '@tremor/react'
import { ValueFormatterParams } from 'ag-grid-community'
import { ChevronRightIcon, Square2StackIcon } from '@heroicons/react/24/outline'
import { useEffect, useState } from 'react'
import { useAtomValue, useSetAtom } from 'jotai'
import clipboardCopy from 'clipboard-copy'
import { Dayjs } from 'dayjs'
import { useNavigate, useSearchParams } from 'react-router-dom'
import Breakdown from '../../../../components/Breakdown'
import {
    useInventoryApiV2AnalyticsSpendCompositionList,
    useInventoryApiV2AnalyticsSpendTableList,
    useInventoryApiV2AnalyticsSpendTrendList,
} from '../../../../api/inventory.gen'
import { isDemoAtom, notificationAtom } from '../../../../store'
import { useIntegrationApiV1ConnectionsSummariesList } from '../../../../api/integration.gen'
import { dateDisplay, dateTimeDisplay } from '../../../../utilities/dateDisplay'
import Spinner from '../../../../components/Spinner'
import DrawerPanel from '../../../../components/DrawerPanel'
import { RenderObject } from '../../../../components/RenderObject'
import { costTrendChart, pieData } from '../../index'
import {
    checkGranularity,
    generateItems,
} from '../../../../utilities/dateComparator'
import { numberDisplay } from '../../../../utilities/numericDisplay'
import SummaryCard from '../../../../components/Cards/SummaryCard'
import { capitalizeFirstLetter } from '../../../../utilities/labelMaker'
import {
    AreaChartIcon,
    BarChartIcon,
    LineChartIcon,
} from '../../../../icons/icons'
import Chart from '../../../../components/Chart'
import { generateVisualMap } from '../../../Assets'
import Table, { IColumn } from '../../../../components/Table'
import { gridOptions, rowGenerator } from '../../Details/Tabs/Metrics'
import { GithubComKaytuIoKaytuEnginePkgInventoryApiSpendTableRow } from '../../../../api/api'
import { searchAtom } from '../../../../utilities/urlstate'

interface ISingle {
    activeTimeRange: { start: Dayjs; end: Dayjs }
    id: string | undefined
}

export default function SingleSpendConnection({
    activeTimeRange,
    id,
}: ISingle) {
    const isDemo = useAtomValue(isDemoAtom)

    const [openDrawer, setOpenDrawer] = useState(false)
    const [selectedChartIndex, setSelectedChartIndex] = useState(0)
    const [selectedChart, setSelectedChart] = useState<'line' | 'bar'>('line')
    const [selectedDatapoint, setSelectedDatapoint] = useState<any>(undefined)
    const [chartAggregation, setChartAggregation] = useState<
        'cumulative' | 'trend'
    >('trend')

    const [selectedGranularity, setSelectedGranularity] = useState<
        'monthly' | 'daily' | 'yearly'
    >(
        checkGranularity(activeTimeRange.start, activeTimeRange.end).monthly
            ? 'monthly'
            : 'daily'
    )
    useEffect(() => {
        setSelectedGranularity(
            checkGranularity(activeTimeRange.start, activeTimeRange.end).monthly
                ? 'monthly'
                : 'daily'
        )
    }, [activeTimeRange])

    const navigate = useNavigate()
    const searchParams = useAtomValue(searchAtom)
    const setNotification = useSetAtom(notificationAtom)

    useEffect(() => {
        if (selectedChartIndex === 0) {
            setSelectedChart('line')
            setChartAggregation('cumulative')
        }
        if (selectedChartIndex === 1) {
            setSelectedChart('line')
            setChartAggregation('trend')
        }
        if (selectedChartIndex === 2) {
            setSelectedChart('bar')
            setChartAggregation('trend')
        }
    }, [selectedChartIndex])

    const { response: costTrend, isLoading: costTrendLoading } =
        useInventoryApiV2AnalyticsSpendTrendList({
            startTime: activeTimeRange.start.unix(),
            endTime: activeTimeRange.end.unix(),
            granularity: selectedGranularity,
            connectionId: [String(id)],
        })

    const tableQuery = (): {
        startTime?: number | undefined
        endTime?: number | undefined
        granularity?: 'daily' | 'monthly' | 'yearly' | undefined
        dimension?: 'metric' | 'connection' | undefined
        connectionId?: string[]
    } => {
        let gra: 'monthly' | 'daily' | 'yearly' = 'daily'
        if (selectedGranularity === 'monthly') {
            gra = 'monthly'
        }

        return {
            startTime: activeTimeRange.start.unix(),
            endTime: activeTimeRange.end.unix(),
            dimension: 'metric',
            granularity: gra,
            connectionId: [String(id)],
        }
    }
    const { response, isLoading } = useInventoryApiV2AnalyticsSpendTableList(
        tableQuery()
    )

    const query = {
        ...(id && {
            connectionId: [id],
        }),
        ...(activeTimeRange.start && {
            startTime: activeTimeRange.start.unix(),
        }),
        ...(activeTimeRange.end && {
            endTime: activeTimeRange.end.unix(),
        }),
    }

    const { response: composition, isLoading: compositionLoading } =
        useInventoryApiV2AnalyticsSpendCompositionList({
            ...query,
            top: 4,
        })
    const { response: accountInfo, isLoading: accountInfoLoading } =
        useIntegrationApiV1ConnectionsSummariesList({
            ...query,
            pageSize: 1,
        })
    const connection = accountInfo?.connections?.at(0)

    const columnGenerator = (
        input:
            | GithubComKaytuIoKaytuEnginePkgInventoryApiSpendTableRow[]
            | undefined
    ) => {
        let columns: IColumn<any, any>[] = []
        if (input) {
            const columnNames =
                input
                    ?.map((row) => {
                        if (row.costValue) {
                            return Object.entries(row.costValue).map(
                                (value) => value[0]
                            )
                        }
                        return []
                    })
                    .flat() || []
            const dynamicCols: IColumn<any, any>[] = columnNames
                .filter((value, index, array) => array.indexOf(value) === index)
                .map((colName) => {
                    const v: IColumn<any, any> = {
                        field: colName,
                        headerName: colName,
                        type: 'string',
                        width: 130,
                        sortable: true,
                        suppressMenu: true,
                        resizable: true,
                        pivot: false,
                        aggFunc: 'sum',
                        valueFormatter: (param: ValueFormatterParams) =>
                            numberDisplay(
                                param.value === undefined ? 0 : param.value,
                                2
                            ),
                    }
                    return v
                })
            columns = [...dynamicCols]
        }
        return columns
    }

    const columns: IColumn<any, any>[] = [
        // ...defaultColumns,
        {
            field: 'category',
            headerName: 'Category',
            type: 'string',
            width: 130,
            filter: true,
            enableRowGroup: true,
            sortable: true,
            resizable: true,
            pinned: true,
        },
        {
            field: 'percent',
            headerName: '%',
            type: 'number',
            width: 90,
            pinned: true,
            sortable: true,
            aggFunc: 'sum',
            resizable: true,
            valueFormatter: (param: ValueFormatterParams) => {
                return param.value ? `${param.value.toFixed(2)}%` : ''
            },
        },
        ...columnGenerator(response),
    ]

    return (
        <>
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
                                <Spinner className="mt-28" />
                            ) : (
                                <List className="mt-2">
                                    <ListItem>
                                        <Text>Cloud Provider</Text>
                                        <Text className="text-gray-800">
                                            {connection?.connector}
                                        </Text>
                                    </ListItem>
                                    <ListItem>
                                        <Text>Discovered name</Text>
                                        <Flex className="gap-1 w-fit">
                                            <Button
                                                variant="light"
                                                onClick={() =>
                                                    clipboardCopy(
                                                        `Discovered name: ${connection?.providerConnectionName}`
                                                    ).then(() =>
                                                        setNotification({
                                                            text: 'Discovered name copied to clipboard',
                                                            type: 'info',
                                                        })
                                                    )
                                                }
                                                icon={Square2StackIcon}
                                            />
                                            <Text
                                                className={`${
                                                    isDemo ? 'blur-sm' : ''
                                                } text-gray-800`}
                                            >
                                                {
                                                    connection?.providerConnectionName
                                                }
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
                                                        `Discovered ID: ${connection?.providerConnectionID}`
                                                    ).then(() =>
                                                        setNotification({
                                                            text: 'Discovered ID copied to clipboard',
                                                            type: 'info',
                                                        })
                                                    )
                                                }
                                                icon={Square2StackIcon}
                                            />
                                            <Text
                                                className={`${
                                                    isDemo ? 'blur-sm' : ''
                                                } text-gray-800`}
                                            >
                                                {
                                                    connection?.providerConnectionID
                                                }
                                            </Text>
                                        </Flex>
                                    </ListItem>
                                    <ListItem>
                                        <Text>Lifecycle state</Text>
                                        <Text className="text-gray-800">
                                            {connection?.lifecycleState}
                                        </Text>
                                    </ListItem>
                                    <ListItem>
                                        <Text>Onboard date</Text>
                                        <Text className="text-gray-800">
                                            {dateTimeDisplay(
                                                connection?.onboardDate
                                            )}
                                        </Text>
                                    </ListItem>
                                    <ListItem>
                                        <Text>Last inventory</Text>
                                        <Text className="text-gray-800">
                                            {dateTimeDisplay(
                                                connection?.lastInventory
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
                            <RenderObject obj={connection} />
                        </DrawerPanel>
                    </Flex>
                </Card>
                <Breakdown
                    chartData={pieData(composition)}
                    loading={compositionLoading}
                />
            </Grid>
            <TabGroup className="mt-4">
                <TabList className="mb-3">
                    <Tab>Trend</Tab>
                    <Tab>Details</Tab>
                </TabList>
                <TabPanels>
                    <TabPanel>
                        <Card>
                            <Grid numItems={6} className="gap-4">
                                <SummaryCard
                                    title=""
                                    metric={accountInfo?.totalCost}
                                    loading={accountInfoLoading}
                                    border={false}
                                    isPrice
                                    isExact
                                />
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
                                            index={selectedChartIndex}
                                            onIndexChange={
                                                setSelectedChartIndex
                                            }
                                            className="w-fit rounded-lg"
                                        >
                                            <TabList variant="solid">
                                                <Tab value="area">
                                                    <AreaChartIcon className="h-5" />
                                                </Tab>
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
                            {costTrend
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
                                <div className="h-2.5 w-2.5 rounded-full bg-openg-950" />
                                {chartAggregation === 'cumulative' ? (
                                    <Text>Accumulated spend</Text>
                                ) : (
                                    <Text>Spend</Text>
                                )}
                            </Flex>
                            <Chart
                                labels={
                                    costTrendChart(
                                        costTrend,
                                        chartAggregation,
                                        'basic',
                                        selectedGranularity
                                    ).label
                                }
                                chartData={
                                    costTrendChart(
                                        costTrend,
                                        chartAggregation,
                                        'basic',
                                        selectedGranularity
                                    ).data
                                }
                                chartType={selectedChart}
                                chartAggregation={chartAggregation}
                                isCost
                                loading={costTrendLoading}
                                visualMap={
                                    chartAggregation === 'cumulative'
                                        ? undefined
                                        : generateVisualMap(
                                              costTrendChart(
                                                  costTrend,
                                                  chartAggregation,
                                                  'basic',
                                                  selectedGranularity
                                              ).flag,
                                              costTrendChart(
                                                  costTrend,
                                                  chartAggregation,
                                                  'basic',
                                                  selectedGranularity
                                              ).label
                                          ).visualMap
                                }
                                markArea={
                                    chartAggregation === 'cumulative'
                                        ? undefined
                                        : generateVisualMap(
                                              costTrendChart(
                                                  costTrend,
                                                  chartAggregation,
                                                  'basic',
                                                  selectedGranularity
                                              ).flag,
                                              costTrendChart(
                                                  costTrend,
                                                  chartAggregation,
                                                  'basic',
                                                  selectedGranularity
                                              ).label
                                          ).markArea
                                }
                                onClick={
                                    chartAggregation === 'cumulative'
                                        ? undefined
                                        : (p) => setSelectedDatapoint(p)
                                }
                            />
                        </Card>
                    </TabPanel>
                    <TabPanel>
                        <Table
                            title="Services"
                            downloadable
                            id="single_spend_service_table"
                            loading={isLoading}
                            columns={columns}
                            rowData={
                                rowGenerator(response, undefined, isLoading)
                                    .finalRow
                            }
                            pinnedRow={
                                rowGenerator(response, undefined, isLoading)
                                    .pinnedRow
                            }
                            options={gridOptions}
                            onRowClicked={(event) => {
                                if (event.data.category.length) {
                                    navigate(
                                        `metric_${event.data.id}?${searchParams}`
                                    )
                                }
                            }}
                            onGridReady={(event) => {
                                if (isLoading) {
                                    event.api.showLoadingOverlay()
                                }
                            }}
                        >
                            <Select
                                enableClear={false}
                                value={selectedGranularity}
                                placeholder={
                                    selectedGranularity
                                        ? capitalizeFirstLetter(
                                              selectedGranularity
                                          )
                                        : ''
                                }
                                onValueChange={(v) => {
                                    // eslint-disable-next-line @typescript-eslint/ban-ts-comment
                                    // @ts-ignore
                                    setSelectedGranularity(v)
                                }}
                                className="w-10"
                            >
                                <SelectItem value="daily">
                                    <Text>Daily</Text>
                                </SelectItem>
                                <SelectItem value="monthly">
                                    <Text>Monthly</Text>
                                </SelectItem>
                            </Select>
                        </Table>
                    </TabPanel>
                </TabPanels>
            </TabGroup>
        </>
    )
}
