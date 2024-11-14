// @ts-nocheck
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
import { GridOptions } from 'ag-grid-community'
import { ChevronRightIcon, Square2StackIcon } from '@heroicons/react/24/outline'
import { useEffect, useState } from 'react'
import { useAtomValue, useSetAtom } from 'jotai'
import clipboardCopy from 'clipboard-copy'
import dayjs, { Dayjs } from 'dayjs'
import { useNavigate, useSearchParams } from 'react-router-dom'
import Breakdown from '../../../../components/Breakdown'
import {
    useInventoryApiV2AnalyticsCompositionDetail,
    useInventoryApiV2AnalyticsMetricList,
    useInventoryApiV2AnalyticsSpendTableList,
    useInventoryApiV2AnalyticsTrendList,
} from '../../../../api/inventory.gen'
import { isDemoAtom, notificationAtom } from '../../../../store'
import Table from '../../../../components/Table'
import { useIntegrationApiV1ConnectionsSummariesList } from '../../../../api/integration.gen'
import { dateTimeDisplay } from '../../../../utilities/dateDisplay'
import Spinner from '../../../../components/Spinner'
import DrawerPanel from '../../../../components/DrawerPanel'
import { RenderObject } from '../../../../components/RenderObject'
import { pieData, resourceTrendChart } from '../../index'
import { checkGranularity } from '../../../../utilities/dateComparator'
import SummaryCard from '../../../../components/Cards/SummaryCard'
import Trends from '../../../../components/Trends'
import {
    defaultColumns,
    rowGenerator,
    SpendrowGenerator,
} from '../../Metric/Table'
import { searchAtom } from '../../../../utilities/urlstate'
import {
    KeyValuePairs,
    Modal,
    Select,
    Tabs,
} from '@cloudscape-design/components'
import KTable from '@cloudscape-design/components/table'
import Box from '@cloudscape-design/components/box'
import SpaceBetween from '@cloudscape-design/components/space-between'
import Badge from '@cloudscape-design/components/badge'
import {
    BreadcrumbGroup,
    DateRangePicker,
    Header,
    Link,
    Pagination,
    PropertyFilter,
} from '@cloudscape-design/components'
import { badgeDelta } from '../../../../utilities/deltaType'
import { numberDisplay } from '../../../../utilities/numericDisplay'

const options: GridOptions = {
    enableGroupEdit: true,
    columnTypes: {
        dimension: {
            enableRowGroup: true,
            enablePivot: true,
        },
    },
    groupDefaultExpanded: -1,
    rowGroupPanelShow: 'always',
    groupAllowUnbalanced: true,
}

interface ISingle {
    activeTimeRange: { start: Dayjs; end: Dayjs }
    id: string | undefined
    resourceId?: string | undefined
}

export default function SingleConnection({
    activeTimeRange,
    id,
    resourceId,
}: ISingle) {
    const isDemo = useAtomValue(isDemoAtom)
    const [openDrawer, setOpenDrawer] = useState(false)
    const setNotification = useSetAtom(notificationAtom)
    const navigate = useNavigate()
    const searchParams = useAtomValue(searchAtom)

    const [selectedGranularity, setSelectedGranularity] = useState<
        'monthly' | 'daily' | 'yearly'
    >(
        checkGranularity(activeTimeRange.start, activeTimeRange.end).daily
            ? 'daily'
            : 'monthly'
    )

    const today = new Date()
    const lastWeek = new Date(
        today.getFullYear(),
        today.getMonth(),
        today.getDate() - 7
    )
    const [granu, setGranu] = useState({
        label: 'Monthly',
        value: 'monthly',
    })
    const [dateInventory, setDateInventory] = useState({
        startDate: activeTimeRange.start?.toISOString(),
        endDate: activeTimeRange.end?.toISOString(),
        type: 'absolute',
    })
    const [dateSpend, setDateSpend] = useState({
        startDate: activeTimeRange.start?.toISOString(),
        endDate: activeTimeRange.end?.toISOString(),
        type: 'absolute',
    })
    const query = {
        ...(id && {
            connectionId: [id],
        }),
        ...(resourceId && {
            resourceCollection: [resourceId],
        }),
        ...(activeTimeRange.start && {
            startTime: activeTimeRange.start.unix(),
        }),
        ...(activeTimeRange.end && {
            endTime: activeTimeRange.end.unix(),
        }),
    }
    useEffect(() => {
        setSelectedGranularity(
            checkGranularity(activeTimeRange.start, activeTimeRange.end).monthly
                ? 'monthly'
                : 'daily'
        )
    }, [activeTimeRange])

    const { response: composition, isLoading: compositionLoading } =
        useInventoryApiV2AnalyticsCompositionDetail('category', {
            ...query,
            top: 4,
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
    const {
        response,
        isLoading,
        sendNowWithParams: sendSpend,
    } = useInventoryApiV2AnalyticsSpendTableList(tableQuery())

    const {
        response: metrics,
        isLoading: metricsLoading,
        sendNowWithParams: sendIntventory,
    } = useInventoryApiV2AnalyticsMetricList({ ...query, pageSize: 1000 })
    const { response: accountInfo, isLoading: accountInfoLoading } =
        useIntegrationApiV1ConnectionsSummariesList({
            ...query,
            pageSize: 1,
            needCost: false,
        })
    const { response: resourceTrend, isLoading: resourceTrendLoading } =
        useInventoryApiV2AnalyticsTrendList({
            ...query,
            // eslint-disable-next-line @typescript-eslint/ban-ts-comment
            // @ts-ignore
            granularity: selectedGranularity,
        })
    const connection = accountInfo?.connections?.at(0)
    const getColumns = () => {
        const temp = []
        temp.push(
            {
                id: 'dimensionName',
                header: 'Service',
                // @ts-ignore
                cell: (item) => item.dimension,
                minWidth: '100px',
                width: '200px',
            },
            {
                id: 'category',
                header: 'Category',
                // @ts-ignore
                cell: (item) => item.category,
                minWidth: '100px',
                width: '200px',
            },
            {
                id: 'percent',
                header: '%',
                // @ts-ignore
                cell: (item) => item.percent.toFixed(2),
                minWidth: '100px',
                width: '100px',
            }
        )
        const input = response
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
            const dynamicCols = columnNames
                .filter((value, index, array) => array.indexOf(value) === index)
                .map((colName) => {
                    const v = {
                        id: colName,
                        header: colName,
                        minWidth: '150px',
                        width: '150px',

                        //  @ts-ignore
                        cell: (item) =>
                            numberDisplay(
                                //  @ts-ignore

                                item[colName] === undefined
                                    ? 0
                                    : //  @ts-ignore

                                      item[colName],
                                2
                            ),
                    }
                    temp.push(v)
                })
        }
        return temp
    }

    const getHeaders = () => {
        const temp = []
        temp.push(
            {
                id: 'dimensionName',
                visible: true,
            },
            {
                id: 'category',
                visible: true,
            }
            // {
            //     id: 'percent',
            //     visible: true,
            // }
        )
        const input = response
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
            const dynamicCols = columnNames
                .filter((value, index, array) => array.indexOf(value) === index)
                .map((colName) => {
                    const v = {
                        id: colName,
                        visible: true,
                    }
                    temp.push(v)
                })
        }
        return temp
    }
    useEffect(() => {
        if (dateInventory.startDate && dateInventory?.endDate) {
            const { startDate, endDate } = dateInventory
            const activeTimeRange = {
                start: dayjs(startDate),
                end: dayjs(endDate),
            }
            const query = {
                ...(id && {
                    connectionId: [id],
                }),
                ...(resourceId && {
                    resourceCollection: [resourceId],
                }),
                ...(activeTimeRange.start && {
                    startTime: activeTimeRange.start.unix(),
                }),
                ...(activeTimeRange.end && {
                    endTime: activeTimeRange.end.unix(),
                }),
            }
            sendIntventory({ ...query, pageSize: 1000 })
        }
    }, [dateInventory])
    useEffect(() => {
        if (dateSpend.startDate && dateSpend?.endDate) {
            const { startDate, endDate } = dateSpend
            const activeTimeRange = {
                start: dayjs(startDate),
                end: dayjs(endDate),
            }

            const query = {
                startTime: activeTimeRange.start.unix(),
                endTime: activeTimeRange.end.unix(),
                dimension: 'metric',
                granularity: granu?.value,
                connectionId: [String(id)],
            }
            sendSpend(query)
        } else {
            const query = {
                // startTime: activeTimeRange.start.unix(),
                // endTime: activeTimeRange.end.unix(),
                dimension: 'metric',
                granularity: granu?.value,
                connectionId: [String(id)],
            }
            sendSpend(query)
        }
    }, [dateSpend, granu])

    return (
        <>
            <Grid numItems={1} className="w-full gap-4 mb-2">
                <Card>
                    <Title className="font-semibold mb-2">
                        Cloud account detail
                    </Title>
                    {accountInfoLoading ? (
                        <Spinner className="mt-28" />
                    ) : (
                        <>
                            <KeyValuePairs
                                columns={4}
                                items={[
                                    {
                                        label: 'Discover name',
                                        value: (
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
                                                <>
                                                    {
                                                        connection?.providerConnectionName
                                                    }
                                                </>
                                            </Flex>
                                        ),
                                    },
                                    {
                                        label: 'Discover ID',
                                        value: (
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
                                                <>
                                                    {
                                                        connection?.providerConnectionID
                                                    }
                                                </>
                                            </Flex>
                                        ),
                                    },

                                    {
                                        label: 'Cloud Provider',
                                        value: connection?.connector,
                                    },
                                    {
                                        label: 'Last Seen',
                                        value: dateTimeDisplay(
                                            connection?.onboardDate
                                        ),
                                    },
                                ]}
                            />
                            {/* <List>
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
                                            {connection?.providerConnectionName}
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
                                            {connection?.providerConnectionID}
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
                            </List> */}
                        </>
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
                    <Modal
                        header="Connection detail"
                        visible={openDrawer}
                        size="large"
                        onDismiss={() => setOpenDrawer(false)}
                    >
                        <RenderObject obj={connection} />
                    </Modal>
                </Card>
                {/* <Breakdown
                    chartData={pieData(composition).newData}
                    oldChartData={pieData(composition).oldData}
                    activeTime={activeTimeRange}
                    loading={compositionLoading}
                /> */}
            </Grid>
            <Tabs
                tabs={[
                    {
                        label: 'Inventory',
                        content: (
                            <>
                                <KTable
                                    className="   min-h-[450px]"
                                    // resizableColumns
                                    // variant="full-page"
                                    renderAriaLive={({
                                        firstIndex,
                                        lastIndex,
                                        totalItemsCount,
                                    }) =>
                                        `Displaying items ${firstIndex} to ${lastIndex} of ${totalItemsCount}`
                                    }
                                    onSortingChange={(event) => {
                                        // setSort(event.detail.sortingColumn.sortingField)
                                        // setSortOrder(!sortOrder)
                                    }}
                                    // sortingColumn={sort}
                                    // sortingDescending={sortOrder}
                                    // sortingDescending={sortOrder == 'desc' ? true : false}
                                    // @ts-ignore
                                    onRowClick={(event) => {
                                        const row = event.detail.item
                                        if (row) {
                                            navigate(
                                                `metric_${row.id}?${searchParams}`
                                            )
                                        }
                                    }}
                                    columnDefinitions={[
                                        {
                                            id: 'connectors',
                                            header: 'Cloud provider',
                                            cell: (item) => item.connectors,
                                            isRowHeader: true,
                                        },
                                        {
                                            id: 'name',
                                            header: 'Metric',
                                            cell: (item) => item.name,
                                        },
                                        {
                                            id: 'category',
                                            header: 'Category',
                                            cell: (item) => item.category,
                                        },
                                        {
                                            id: 'count',
                                            header: 'Resource count',
                                            cell: (item) => item.count,
                                        },
                                        {
                                            id: 'change_percent',
                                            header: 'Change (%)',
                                            cell: (item) => (
                                                <>
                                                    {badgeDelta(
                                                        item?.old_count,
                                                        item?.count
                                                    )}
                                                </>
                                            ),
                                        },
                                        {
                                            id: 'change_delta',
                                            header: 'Change (Δ)',
                                            cell: (item) => (
                                                <>
                                                    {badgeDelta(
                                                        item?.old_count,
                                                        item?.count
                                                    )}
                                                </>
                                            ),
                                        },
                                    ]}
                                    columnDisplay={[
                                        { id: 'connectors', visible: true },
                                        { id: 'name', visible: true },
                                        { id: 'category', visible: true },
                                        {
                                            id: 'count',
                                            visible: true,
                                        },
                                        { id: 'change_percent', visible: true },
                                        { id: 'change_delta', visible: true },

                                        // { id: 'action', visible: true },
                                    ]}
                                    enableKeyboardNavigation
                                    // @ts-ignore
                                    items={rowGenerator(metrics?.metrics || [])}
                                    loading={metricsLoading}
                                    loadingText="Loading resources"
                                    filter={
                                        <DateRangePicker
                                            onChange={({ detail }) =>
                                                // @ts-ignore
                                                setDateInventory(detail.value)
                                            }
                                            // @ts-ignore

                                            value={dateInventory}
                                            dateOnly={true}
                                            rangeSelectorMode={'absolute-only'}
                                            // relativeOptions={[
                                            //     {
                                            //         key: 'previous-5-minutes',
                                            //         amount: 5,
                                            //         unit: 'minute',
                                            //         type: 'relative',
                                            //     },
                                            //     {
                                            //         key: 'previous-30-minutes',
                                            //         amount: 30,
                                            //         unit: 'minute',
                                            //         type: 'relative',
                                            //     },
                                            //     {
                                            //         key: 'previous-1-hour',
                                            //         amount: 1,
                                            //         unit: 'hour',
                                            //         type: 'relative',
                                            //     },
                                            //     {
                                            //         key: 'previous-6-hours',
                                            //         amount: 6,
                                            //         unit: 'hour',
                                            //         type: 'relative',
                                            //     },
                                            //     {
                                            //         key: 'previous-3-days',
                                            //         amount: 3,
                                            //         unit: 'day',
                                            //         type: 'relative',
                                            //     },
                                            //     {
                                            //         key: 'previous-7-days',
                                            //         amount: 7,
                                            //         unit: 'day',
                                            //         type: 'relative',
                                            //     },
                                            // ]}

                                            hideTimeOffset
                                            // showClearButton={false}
                                            absoluteFormat="long-localized"
                                            isValidRange={(range) => {
                                                if (range.type === 'absolute') {
                                                    const [
                                                        startDateWithoutTime,
                                                    ] =
                                                        range.startDate.split(
                                                            'T'
                                                        )
                                                    const [endDateWithoutTime] =
                                                        range.endDate.split('T')
                                                    if (
                                                        !startDateWithoutTime ||
                                                        !endDateWithoutTime
                                                    ) {
                                                        return {
                                                            valid: false,
                                                            errorMessage:
                                                                'The selected date range is incomplete. Select a start and end date for the date range.',
                                                        }
                                                    }
                                                    if (
                                                        new Date(
                                                            range.startDate
                                                        ) -
                                                            new Date(
                                                                range.endDate
                                                            ) >
                                                        0
                                                    ) {
                                                        return {
                                                            valid: false,
                                                            errorMessage:
                                                                'The selected date range is invalid. The start date must be before the end date.',
                                                        }
                                                    }
                                                }
                                                return { valid: true }
                                            }}
                                            i18nStrings={{}}
                                            placeholder="Filter by a date and time range"
                                        />
                                    }
                                    // stickyColumns={{ first: 0, last: 1 }}
                                    // stripedRows
                                    trackBy="id"
                                    empty={
                                        <Box
                                            margin={{ vertical: 'xs' }}
                                            textAlign="center"
                                            color="inherit"
                                        >
                                            <SpaceBetween size="m">
                                                <b>No resources</b>
                                            </SpaceBetween>
                                        </Box>
                                    }
                                    header={
                                        <Header className="w-full">
                                            Resources{' '}
                                        </Header>
                                    }
                                />
                            </>
                        ),
                        id: '0',
                    },
                    {
                        label: 'Spend',
                        content: (
                            <>
                                <KTable
                                    className="   min-h-[450px] max-w-screen-xl"
                                    // resizableColumns
                                    // variant="full-page"
                                    renderAriaLive={({
                                        firstIndex,
                                        lastIndex,
                                        totalItemsCount,
                                    }) =>
                                        `Displaying items ${firstIndex} to ${lastIndex} of ${totalItemsCount}`
                                    }
                                    filter={
                                        <Flex
                                            flexDirection="row"
                                            justifyContent="start"
                                            className="gap-2"
                                        >
                                            <Select
                                                label="Granularity"
                                                className=" min-w-[150px]"
                                                inlineLabelText="Granularity"
                                                selectedOption={granu}
                                                onChange={({ detail }) => {
                                                    setGranu(
                                                        detail.selectedOption
                                                    )
                                                    setSelectedGranularity(
                                                        detail.selectedOption
                                                            .value
                                                    )
                                                }}
                                                options={[
                                                    {
                                                        label: 'Daily',
                                                        value: 'daily',
                                                    },
                                                    {
                                                        label: 'Monthly',
                                                        value: 'monthly',
                                                    },
                                                ]}
                                            />

                                            <DateRangePicker
                                                onChange={({ detail }) =>
                                                    // @ts-ignore
                                                    setDateSpend(detail.value)
                                                }
                                                // @ts-ignore

                                                value={dateSpend}
                                                dateOnly={true}
                                                rangeSelectorMode={
                                                    'absolute-only'
                                                }
                                                // relativeOptions={[
                                                //     {
                                                //         key: 'previous-5-minutes',
                                                //         amount: 5,
                                                //         unit: 'minute',
                                                //         type: 'relative',
                                                //     },
                                                //     {
                                                //         key: 'previous-30-minutes',
                                                //         amount: 30,
                                                //         unit: 'minute',
                                                //         type: 'relative',
                                                //     },
                                                //     {
                                                //         key: 'previous-1-hour',
                                                //         amount: 1,
                                                //         unit: 'hour',
                                                //         type: 'relative',
                                                //     },
                                                //     {
                                                //         key: 'previous-6-hours',
                                                //         amount: 6,
                                                //         unit: 'hour',
                                                //         type: 'relative',
                                                //     },
                                                //     {
                                                //         key: 'previous-3-days',
                                                //         amount: 3,
                                                //         unit: 'day',
                                                //         type: 'relative',
                                                //     },
                                                //     {
                                                //         key: 'previous-7-days',
                                                //         amount: 7,
                                                //         unit: 'day',
                                                //         type: 'relative',
                                                //     },
                                                // ]}

                                                hideTimeOffset
                                                // showClearButton={false}
                                                absoluteFormat="long-localized"
                                                isValidRange={(range) => {
                                                    if (
                                                        range.type ===
                                                        'absolute'
                                                    ) {
                                                        const [
                                                            startDateWithoutTime,
                                                        ] =
                                                            range.startDate.split(
                                                                'T'
                                                            )
                                                        const [
                                                            endDateWithoutTime,
                                                        ] =
                                                            range.endDate.split(
                                                                'T'
                                                            )
                                                        if (
                                                            !startDateWithoutTime ||
                                                            !endDateWithoutTime
                                                        ) {
                                                            return {
                                                                valid: false,
                                                                errorMessage:
                                                                    'The selected date range is incomplete. Select a start and end date for the date range.',
                                                            }
                                                        }
                                                        if (
                                                            new Date(
                                                                range.startDate
                                                            ) -
                                                                new Date(
                                                                    range.endDate
                                                                ) >
                                                            0
                                                        ) {
                                                            return {
                                                                valid: false,
                                                                errorMessage:
                                                                    'The selected date range is invalid. The start date must be before the end date.',
                                                            }
                                                        }
                                                    }
                                                    return { valid: true }
                                                }}
                                                i18nStrings={{}}
                                                placeholder="Filter by a date and time range"
                                            />
                                        </Flex>
                                    }
                                    onSortingChange={(event) => {
                                        // setSort(event.detail.sortingColumn.sortingField)
                                        // setSortOrder(!sortOrder)
                                    }}
                                    // sortingColumn={sort}
                                    // sortingDescending={sortOrder}
                                    // sortingDescending={sortOrder == 'desc' ? true : false}
                                    // @ts-ignore
                                    onRowClick={(event) => {
                                        const row = event.detail.item
                                        if (row) {
                                            navigate(
                                                `metric_${row.id}?${searchParams}`
                                            )
                                        }
                                    }}
                                    resizableColumns={true}
                                    stickyColumns={{ first: 2, last: 0 }}
                                    columnDefinitions={getColumns()}
                                    columnDisplay={getHeaders()}
                                    enableKeyboardNavigation
                                    // @ts-ignore
                                    items={
                                        SpendrowGenerator(
                                            response,
                                            undefined,
                                            isLoading
                                        ).finalRow
                                    }
                                    loading={metricsLoading}
                                    loadingText="Loading resources"
                                    // stickyColumns={{ first: 0, last: 1 }}
                                    // stripedRows
                                    trackBy="id"
                                    empty={
                                        <Box
                                            margin={{ vertical: 'xs' }}
                                            textAlign="center"
                                            color="inherit"
                                        >
                                            <SpaceBetween size="m">
                                                <b>No resources</b>
                                            </SpaceBetween>
                                        </Box>
                                    }
                                    header={
                                        <Header className="w-full">
                                            Cloud Spend{' '}
                                        </Header>
                                    }
                                />
                            </>
                        ),
                        id: '1',
                    },
                ]}
            />
            {/* <TabGroup className="mt-4">
                <TabList className="mb-3">
                    <Tab>Trend</Tab>
                    <Tab>Details</Tab>
                </TabList>
                <TabPanels>
                    <TabPanel>
                        <Trends
                            activeTimeRange={activeTimeRange}
                            trend={resourceTrend}
                            firstKPI={
                                <SummaryCard
                                    title=""
                                    metric={connection?.resourceCount}
                                    metricPrev={connection?.oldResourceCount}
                                    loading={resourceTrendLoading}
                                    border={false}
                                />
                            }
                            trendName="Resources"
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
                            loading={resourceTrendLoading}
                            onGranularityChange={(gra) =>
                                setSelectedGranularity(gra)
                            }
                        />
                    </TabPanel>
                    <TabPanel>
                        <Table
                            options={options}
                            title="Resources"
                            downloadable
                            id="asset_resource_metrics"
                            rowData={rowGenerator(metrics?.metrics || [])}
                            columns={[
                                ...defaultColumns,
                                {
                                    field: 'category',
                                    enableRowGroup: true,
                                    headerName: 'Category',
                                    resizable: true,
                                    sortable: true,
                                    filter: true,
                                    type: 'string',
                                },
                            ]}
                            loading={metricsLoading}
                            onRowClicked={(e) => {
                                if (e.data) {
                                    navigate(
                                        `metric_${e.data.id}?${searchParams}`
                                    )
                                }
                            }}
                        />
                    </TabPanel>
                </TabPanels>
            </TabGroup> */}
        </>
    )
}
