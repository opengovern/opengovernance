import {
    GridOptions,
    ValueFormatterParams,
    IAggFuncParams,
    ICellRendererParams,
} from 'ag-grid-community'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { Dispatch, SetStateAction, useEffect, useState } from 'react'
import {
    ArrowTrendingDownIcon,
    ArrowTrendingUpIcon,
    CloudIcon,
    CurrencyDollarIcon,
    ListBulletIcon,
    Squares2X2Icon,
} from '@heroicons/react/24/outline'
import dayjs, { Dayjs } from 'dayjs'
import { Flex, Text } from '@tremor/react'
import { GithubComKaytuIoKaytuEnginePkgInventoryApiSpendTableRow } from '../../../api/api'
import AdvancedTable, { IColumn } from '../../../components/AdvancedTable'
import {
    exactPriceDisplay,
    numberDisplay,
} from '../../../utilities/numericDisplay'
import { renderDateText } from '../../../components/Layout/Header/DatePicker'

type MSort = {
    sortCol: string
    sortType: 'asc' | 'desc' | null
}

interface IMetricTable {
    timeRange: { start: Dayjs; end: Dayjs }
    prevTimeRange: { start: Dayjs; end: Dayjs }
    selectedGranularity: 'monthly' | 'daily'
    onGranularityChange: Dispatch<SetStateAction<'monthly' | 'daily'>>
    response:
        | GithubComKaytuIoKaytuEnginePkgInventoryApiSpendTableRow[]
        | undefined
    responsePrev:
        | GithubComKaytuIoKaytuEnginePkgInventoryApiSpendTableRow[]
        | undefined
    isLoading: boolean
}

const rowGenerator = (
    input:
        | GithubComKaytuIoKaytuEnginePkgInventoryApiSpendTableRow[]
        | undefined,
    inputPrev:
        | GithubComKaytuIoKaytuEnginePkgInventoryApiSpendTableRow[]
        | undefined,
    loading: boolean
) => {
    let sum = 0
    let sumInPrev = 0
    const roww = []
    const granularity: any = {}
    let pinnedRow = [
        { totalCost: sum, dimension: 'Total spend', ...granularity },
    ]
    if (!loading) {
        const rows =
            input?.map((row) => {
                let temp: [string, number][] = []
                let totalCost = 0
                if (row.costValue) {
                    temp = Object.entries(row.costValue)
                }
                temp.forEach((v) => {
                    totalCost += v[1]
                })
                const dateColumns = Object.fromEntries(temp)
                const totalMetricSpendInPrev =
                    inputPrev
                        ?.flatMap((v) => Object.entries(v.costValue || {}))
                        .map((v) => v[1])
                        .reduce((prev, curr) => prev + curr, 0) || 0
                const totalSpendInPrev =
                    inputPrev
                        ?.filter((v) => v.dimensionId === row.dimensionId)
                        .flatMap((v) => Object.entries(v.costValue || {}))
                        .map((v) => v[1])
                        .reduce((prev, curr) => prev + curr, 0) || 0
                const result = {
                    dimension: row.dimensionName
                        ? row.dimensionName
                        : row.dimensionId,
                    dimensionId: row.dimensionId,
                    category: row.category,
                    accountId: row.accountID,
                    connector: row.connector,
                    id: row.dimensionId,
                    totalCost,
                    prevTotalCost: totalSpendInPrev,
                    prevPercent:
                        (totalSpendInPrev / totalMetricSpendInPrev) * 100.0,
                    changePercent:
                        ((totalCost - totalSpendInPrev) / totalSpendInPrev) *
                        100.0,
                    change: totalCost - totalSpendInPrev,
                    ...dateColumns,
                }
                return result
            }) || []
        for (let i = 0; i < rows.length; i += 1) {
            sum += rows[i].totalCost
            sumInPrev += rows[i].prevTotalCost
            // eslint-disable-next-line array-callback-return
            Object.entries(rows[i]).map(([key, value]) => {
                if (Number(key[0])) {
                    if (granularity[key]) {
                        granularity[key] += value
                    } else {
                        granularity[key] = value
                    }
                }
            })
        }
        pinnedRow = [
            {
                totalCost: sum,
                percent: 100.0,
                prevTotalCost: sumInPrev,
                prevPercent: 100.0,
                changePercent: ((sum - sumInPrev) / sumInPrev) * 100.0,
                change: sum - sumInPrev,
                dimension: 'Total spend',
                ...granularity,
            },
        ]
        for (let i = 0; i < rows.length; i += 1) {
            roww.push({
                ...rows[i],
                percent: (rows[i].totalCost / sum) * 100,
            })
        }
    }
    const finalRow = roww.sort((a, b) => b.totalCost - a.totalCost)
    return {
        finalRow,
        pinnedRow,
    }
}

export const gridOptions: GridOptions = {
    columnTypes: {
        dimension: {
            enableRowGroup: true,
            enablePivot: true,
        },
    },
    rowGroupPanelShow: 'always',
    groupAllowUnbalanced: true,
    autoGroupColumnDef: {
        pinned: true,
        width: 150,
        suppressMenu: true,
        sortable: true,
        filter: true,
        resizable: true,
        cellRendererParams: {
            footerValueGetter: (params: any) => {
                const isRootLevel = params.node.level === -1
                if (isRootLevel) {
                    return 'Grand Total'
                }
                return `Sub Total (${params.value})`
            },
        },
    },
    enableRangeSelection: true,
    // groupIncludeFooter: true,
    // groupIncludeTotalFooter: true,
}

export default function MetricTable({
    timeRange,
    prevTimeRange,
    response,
    responsePrev,
    isLoading,
    selectedGranularity,
    onGranularityChange,
}: IMetricTable) {
    const navigate = useNavigate()
    const [searchParams, setSearchParams] = useSearchParams()

    const columnGenerator = (
        input:
            | GithubComKaytuIoKaytuEnginePkgInventoryApiSpendTableRow[]
            | undefined
    ) => {
        let columns: IColumn<
            GithubComKaytuIoKaytuEnginePkgInventoryApiSpendTableRow,
            any
        >[] = []
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
                .map((colName, idx) => {
                    const v: IColumn<any, any> = {
                        field: colName,
                        headerName: colName,
                        type: 'string',
                        width: 130,
                        aggFunc: 'sum',
                        filter: true,
                        sortable: true,
                        resizable: true,
                        suppressMenu: true,
                        columnGroupShow: 'open',
                        valueFormatter: (
                            param: ValueFormatterParams<
                                GithubComKaytuIoKaytuEnginePkgInventoryApiSpendTableRow,
                                any
                            >
                        ) => {
                            return exactPriceDisplay(
                                param.value === undefined ? 0 : param.value
                            )
                        },
                    }
                    return v
                })

            const total: IColumn<any, any> = {
                field: 'totalCost',
                headerName: 'Total',
                type: 'price',
                width: 130,
                aggFunc: 'sum',
                filter: true,
                sortable: true,
                resizable: true,
                suppressMenu: true,
                columnGroupShow: 'closed',
                valueFormatter: (param: ValueFormatterParams) =>
                    exactPriceDisplay(param.value),
            }

            columns = [total, ...dynamicCols]
        }
        return columns
    }

    const columns: IColumn<any, any>[] = [
        {
            headerName: 'Metric Metadata',
            type: 'parent',
            pinned: true,
            children: [
                {
                    field: 'category',
                    headerName: 'Category',
                    type: 'string',
                    width: 110,
                    filter: true,
                    sortable: true,
                    resizable: true,
                    suppressMenu: true,
                    enableRowGroup: true,
                    pinned: true,
                },
                {
                    field: 'connector',
                    headerName: 'Provider',
                    type: 'string',
                    width: 100,
                    enableRowGroup: true,
                    filter: true,
                    sortable: true,
                    resizable: true,
                    suppressMenu: true,
                    pinned: true,
                },
                {
                    field: 'dimension',
                    headerName: 'Name',
                    type: 'string',
                    width: 230,
                    filter: true,
                    sortable: true,
                    resizable: true,
                    suppressMenu: true,
                    pinned: true,
                },
            ],
        },
        {
            headerName: `Current Period`,
            type: 'parent',
            pinned: true,
            wrapHeaderText: true,
            autoHeaderHeight: true,
            children: [
                {
                    field: 'totalCost',
                    headerName: 'Spend',
                    type: 'price',
                    aggFunc: 'sum',
                    filter: true,
                    sortable: true,
                    resizable: true,
                    suppressMenu: true,
                    width: 200,
                    valueFormatter: (param: ValueFormatterParams) =>
                        exactPriceDisplay(param.value),
                },
                // {
                //     field: 'percent',
                //     headerName: '% of Total',
                //     type: 'string',
                //     aggFunc: 'sum',
                //     filter: true,
                //     sortable: true,
                //     resizable: true,
                //     suppressMenu: true,
                //     width: 100,
                //     valueFormatter: (param: ValueFormatterParams) =>
                //         `${numberDisplay(param.value)}%`,
                // },
            ],
        },
        {
            headerName: `Previous Period [${renderDateText(
                prevTimeRange.start,
                prevTimeRange.end
            )}]`,
            type: 'parent',
            pinned: true,
            wrapHeaderText: true,
            autoHeaderHeight: true,
            children: [
                {
                    field: 'prevTotalCost',
                    headerName: 'Spend',
                    type: 'string',
                    aggFunc: 'sum',
                    filter: true,
                    sortable: true,
                    resizable: true,
                    suppressMenu: true,
                    width: 200,
                    valueFormatter: (param: ValueFormatterParams) =>
                        exactPriceDisplay(param.value),
                },
                // {
                //     field: 'prevPercent',
                //     headerName: '% of Total',
                //     type: 'string',
                //     aggFunc: 'sum',
                //     filter: true,
                //     sortable: true,
                //     resizable: true,
                //     suppressMenu: true,
                //     width: 100,
                //     valueFormatter: (param: ValueFormatterParams) =>
                //         `${numberDisplay(param.value)}%`,
                // },
            ],
        },
        {
            headerName: 'Change',
            type: 'parent',
            wrapHeaderText: true,
            autoHeaderHeight: true,
            pinned: true,
            children: [
                {
                    field: 'changePercent',
                    headerName: '%',
                    type: 'string',
                    width: 110,
                    filter: true,
                    sortable: true,
                    resizable: true,
                    suppressMenu: true,
                    // eslint-disable-next-line react/no-unstable-nested-components
                    cellRenderer: (
                        param: ICellRendererParams<
                            GithubComKaytuIoKaytuEnginePkgInventoryApiSpendTableRow,
                            any
                        >
                    ) => {
                        return (
                            <Flex
                                flexDirection="row"
                                justifyContent="start"
                                alignItems="center"
                                className={`h-full w-full space-x-1 ${
                                    param.value > 0
                                        ? 'text-green-600'
                                        : 'text-red-600'
                                }`}
                            >
                                {param.value > 0 ? (
                                    <ArrowTrendingUpIcon className="w-4" />
                                ) : (
                                    <ArrowTrendingDownIcon className="w-4" />
                                )}

                                <Text
                                    className={
                                        param.value > 0
                                            ? 'text-green-600'
                                            : 'text-red-600'
                                    }
                                >
                                    {Math.abs(param.value) > 1000 ? '+' : ''}
                                    {numberDisplay(
                                        Math.min(1000, Math.abs(param.value)),
                                        0
                                    )}
                                    %
                                </Text>
                            </Flex>
                        )
                    },
                },
                {
                    field: 'change',
                    headerName: 'Delta',
                    type: 'string',
                    width: 100,
                    aggFunc: 'sum',
                    filter: true,
                    sortable: true,
                    resizable: true,
                    suppressMenu: true,
                    // eslint-disable-next-line react/no-unstable-nested-components
                    cellRenderer: (
                        param: ICellRendererParams<
                            GithubComKaytuIoKaytuEnginePkgInventoryApiSpendTableRow,
                            any
                        >
                    ) => {
                        return (
                            <Flex
                                flexDirection="row"
                                justifyContent="start"
                                alignItems="center"
                                className={`h-full w-full space-x-1 ${
                                    param.value > 0
                                        ? 'text-green-600'
                                        : 'text-red-600'
                                }`}
                            >
                                {param.value > 0 ? (
                                    <ArrowTrendingUpIcon className="w-4" />
                                ) : (
                                    <ArrowTrendingDownIcon className="w-4" />
                                )}

                                <Text
                                    className={
                                        param.value > 0
                                            ? 'text-green-600'
                                            : 'text-red-600'
                                    }
                                >
                                    ${numberDisplay(Math.abs(param.value), 0)}
                                </Text>
                            </Flex>
                        )
                    },
                },
            ],
        },
        {
            headerName: 'Granular Details',
            type: 'parent',
            wrapHeaderText: true,
            autoHeaderHeight: true,
            children: [...columnGenerator(response)],
        },
    ]

    const [manualTableSort, onManualSortChange] = useState<MSort>({
        sortCol: 'none',
        sortType: null,
    })

    const [manualGrouping, onManualGrouping] = useState<string>(
        searchParams.get('groupby') === 'category' ? 'category' : 'none'
    )

    const filterTabs = [
        {
            type: 0,
            icon: CurrencyDollarIcon,
            name: 'Sort by Spend',
            function: () => {
                onManualSortChange({
                    sortCol: 'totalCost',
                    sortType: 'desc',
                })
                onManualGrouping('none')
            },
        },
        {
            type: 1,
            icon: ListBulletIcon,
            name: 'Sort by Change',
            function: () => {
                onManualSortChange({
                    sortCol: 'change',
                    sortType: 'desc',
                })
                onManualGrouping('none')
            },
        },
        {
            type: 2,
            icon: Squares2X2Icon,
            name: 'Group by Metric Category',
            function: () => {
                onManualGrouping('category')
                onManualSortChange({
                    sortCol: 'totalCost',
                    sortType: 'desc',
                })
            },
        },
        {
            type: 3,
            icon: CloudIcon,
            name: 'Group by Provider',
            function: () => {
                onManualGrouping('connector')
                onManualSortChange({
                    sortCol: 'totalCost',
                    sortType: 'desc',
                })
            },
        },
    ]

    const [tab, setTab] = useState(
        searchParams.get('groupby') === 'category' ? 2 : 0
    )

    const [tableKey, setTableKey] = useState('')

    useEffect(() => {
        setTableKey(Math.random().toString(16).slice(2, 8))
    }, [manualGrouping, timeRange, response])

    return (
        <AdvancedTable
            key={`metric_${tableKey}`}
            title="Metric list"
            downloadable
            id="spend_service_table"
            loading={isLoading}
            columns={columns}
            rowData={rowGenerator(response, responsePrev, isLoading).finalRow}
            pinnedRow={
                rowGenerator(response, responsePrev, isLoading).pinnedRow
            }
            options={gridOptions}
            onRowClicked={(event) => {
                if (event.data.category.length) {
                    navigate(`metric_${event.data.id}?${searchParams}`)
                }
            }}
            onGridReady={(event) => {
                if (isLoading) {
                    event.api.showLoadingOverlay()
                }
            }}
            selectedGranularity={selectedGranularity}
            onGranularityChange={onGranularityChange}
            manualSort={manualTableSort}
            manualGrouping={manualGrouping}
            filterTabs={filterTabs}
            tabIdx={tab}
            setTabIdx={setTab}
        />
    )
}
