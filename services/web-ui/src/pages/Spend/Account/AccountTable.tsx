import {
    GridOptions,
    ICellRendererParams,
    ValueFormatterParams,
} from 'ag-grid-community'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { Dispatch, SetStateAction, useEffect, useState } from 'react'
import {
    CurrencyDollarIcon,
    ListBulletIcon,
    ArrowTrendingUpIcon,
    CloudIcon,
    ArrowTrendingDownIcon,
} from '@heroicons/react/24/outline'
import dayjs, { Dayjs } from 'dayjs'
import { useAtomValue } from 'jotai'
import { Flex, Text } from '@tremor/react'
import { GithubComKaytuIoKaytuEnginePkgInventoryApiSpendTableRow } from '../../../api/api'
import AdvancedTable, { IColumn } from '../../../components/AdvancedTable'
import {
    exactPriceDisplay,
    numberDisplay,
} from '../../../utilities/numericDisplay'
import { renderDateText } from '../../../components/Layout/Header/DatePicker'
import { searchAtom } from '../../../utilities/urlstate'
import { isDemoAtom } from '../../../store'

export type MSort = {
    sortCol: string
    sortType: 'asc' | 'desc' | null
}

interface IAccountTable {
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
    ref?: React.MutableRefObject<any>
}

export const pickFromRecord = (
    v: Record<string, number> | undefined,
    item: 'oldest' | 'latest'
) => {
    if (v === undefined) {
        return 0
    }
    const m = Object.entries(v)
        .map((i) => {
            return {
                date: dayjs(i[0]),
                value: i[1],
            }
        })
        .sort((a, b) => {
            if (a.date.isSame(b.date)) {
                return 0
            }
            return a.date.isAfter(b.date) ? 1 : -1
        })

    const idx = item === 'oldest' ? 0 : m.length - 1
    const res = m.at(idx)?.value || 0
    return res
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
                Object.values(temp).forEach((v) => {
                    totalCost += v[1]
                })
                const dateColumns = Object.fromEntries(temp)
                const totalAccountsSpendInPrev =
                    inputPrev
                        ?.flatMap((v) => Object.entries(v.costValue || {}))
                        .map((v) => v[1])
                        .reduce((prev, curr) => prev + curr, 0) || 0
                const totalSpendInPrev =
                    inputPrev
                        ?.filter((v) => v.accountID === row.accountID)
                        .flatMap((v) => Object.entries(v.costValue || {}))
                        .map((v) => v[1])
                        .reduce((prev, curr) => prev + curr, 0) || 0

                return {
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
                        (totalSpendInPrev / totalAccountsSpendInPrev) * 100.0,
                    changePercent:
                        ((totalCost - totalSpendInPrev) / totalSpendInPrev) *
                        100.0,
                    change: totalCost - totalSpendInPrev,
                    ...dateColumns,
                }
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
                spendInPrev: 0,
            })
        }
    }
    const finalRow = roww.sort((a, b) => b.totalCost - a.totalCost)
    return {
        finalRow,
        pinnedRow,
    }
}

const gridOptions: GridOptions = {
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
    maintainColumnOrder: true,
}

export default function AccountTable({
    timeRange,
    prevTimeRange,
    selectedGranularity,
    onGranularityChange,
    response,
    responsePrev,
    isLoading,
    ref,
}: IAccountTable) {
    const navigate = useNavigate()
    const searchParams = useAtomValue(searchAtom)

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
                input?.flatMap((row) => {
                    if (row.costValue) {
                        return Object.entries(row.costValue).map(
                            (value) => value[0]
                        )
                    }
                    return []
                }) || []
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
                        columnGroupShow: 'open',
                        valueFormatter: (param: ValueFormatterParams) =>
                            exactPriceDisplay(
                                param.value === undefined ? 0 : param.value
                            ),
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
    const [manualGrouping, onManualGrouping] = useState<string>('none')
    const isDemo = useAtomValue(isDemoAtom)

    const columns: IColumn<any, any>[] = [
        {
            field: 'connector',
            headerName: 'Provider',
            type: 'string',
            width: 90,
            suppressMenu: true,
            enableRowGroup: true,
            rowGroup: manualGrouping === 'connector',
            pinned: true,
            filter: true,
            resizable: true,
            sortable: true,
        },
        {
            headerName: 'Account Information',
            type: 'string',
            children: [
                {
                    field: 'dimension',
                    headerName: 'Discovered Name',
                    type: 'string',
                    pinned: true,
                    width: 200,
                    filter: true,
                    sortable: true,
                    resizable: true,
                    suppressMenu: true,
                    // eslint-disable-next-line react/no-unstable-nested-components
                    cellRenderer: (param: ValueFormatterParams) => (
                        <span className={isDemo ? 'blur-sm' : ''}>
                            {param.value}
                        </span>
                    ),
                },
                {
                    field: 'accountId',
                    headerName: 'Discovered ID',
                    type: 'string',
                    width: 150,
                    pinned: true,
                    filter: true,
                    sortable: true,
                    resizable: true,
                    suppressMenu: true,
                    // eslint-disable-next-line react/no-unstable-nested-components
                    cellRenderer: (param: ValueFormatterParams) => (
                        <span className={isDemo ? 'blur-sm' : ''}>
                            {param.value}
                        </span>
                    ),
                },
                {
                    field: 'dimensionId',
                    headerName: 'OpenGovernance Connection ID',
                    type: 'string',
                    width: 150,
                    pinned: true,
                    filter: true,
                    sortable: true,
                    resizable: true,
                    suppressMenu: true,
                    hide: true,
                },
            ],
        },
        {
            headerName: `Current Period`,
            type: 'string',
            children: [
                {
                    field: 'totalCost',
                    headerName: 'Spend',
                    type: 'price',
                    width: 200,
                    aggFunc: 'sum',
                    filter: true,
                    sortable: true,
                    resizable: true,
                    suppressMenu: true,
                    valueFormatter: (param: ValueFormatterParams) =>
                        exactPriceDisplay(param.value),
                },
                // {
                //     field: 'percent',
                //     headerName: '% of Total',
                //     type: 'string',
                //     width: 100,
                //     aggFunc: 'sum',
                //     filter: true,
                //     sortable: true,
                //     resizable: true,
                //     suppressMenu: true,
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
            type: 'string',
            children: [
                {
                    field: 'prevTotalCost',
                    headerName: 'Spend',
                    type: 'string',
                    width: 200,
                    aggFunc: 'sum',
                    filter: true,
                    sortable: true,
                    resizable: true,
                    suppressMenu: true,
                    valueFormatter: (param: ValueFormatterParams) =>
                        exactPriceDisplay(param.value),
                },
                // {
                //     field: 'prevPercent',
                //     headerName: '% of Total',
                //     type: 'string',
                //     width: 100,
                //     aggFunc: 'sum',
                //     filter: true,
                //     sortable: true,
                //     resizable: true,
                //     suppressMenu: true,
                //     valueFormatter: (param: ValueFormatterParams) =>
                //         `${numberDisplay(param.value)}%`,
                // },
            ],
        },
        {
            headerName: 'Change',
            type: 'string',
            children: [
                {
                    field: 'changePercent',
                    headerName: '%',
                    type: 'string',
                    width: 110,
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
            type: 'string',
            children: [...columnGenerator(response)],
        },
    ]

    const [manualTableSort, onManualSortChange] = useState<MSort>({
        sortCol: 'none',
        sortType: null,
    })

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
            icon: ArrowTrendingUpIcon,
            name: 'Sort by Account Name',
            function: () => {
                onManualSortChange({
                    sortCol: 'dimension',
                    sortType: 'desc',
                })
                onManualGrouping('none')
            },
        },
        {
            type: 3,
            icon: CloudIcon,
            name: 'Group by Provider',
            function: () => {
                onManualGrouping('connector')
                onManualSortChange({
                    sortCol: 'none',
                    sortType: null,
                })
            },
        },
    ]
    const [tab, setTab] = useState(0)
    const [tableKey, setTableKey] = useState('')

    useEffect(() => {
        setTableKey(Math.random().toString(16).slice(2, 8))
    }, [manualGrouping, timeRange, response])

    return (
        <AdvancedTable
            key={`account_${tableKey}`}
            title="Cloud account list"
            downloadable
            id="spend_connection_table"
            loading={isLoading}
            columns={columns}
            rowData={rowGenerator(response, responsePrev, isLoading).finalRow}
            pinnedRow={
                rowGenerator(response, responsePrev, isLoading).pinnedRow
            }
            options={gridOptions}
            onRowClicked={(event) => {
                if (event.data.id) {
                    navigate(`account_${event.data.id}?${searchParams}`)
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
            ref={ref}
        />
    )
}
