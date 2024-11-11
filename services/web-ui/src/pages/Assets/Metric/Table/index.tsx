import { Dayjs } from 'dayjs'
import { GridOptions, ICellRendererParams } from 'ag-grid-community'
import {
    ArrowTrendingUpIcon,
    CircleStackIcon,
    CloudIcon,
    ListBulletIcon,
    Squares2X2Icon,
} from '@heroicons/react/24/outline'
import { useEffect, useState } from 'react'
import { useSetAtom } from 'jotai/index'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { notificationAtom } from '../../../../store'
import { IColumn } from '../../../../components/Table'
import { GithubComKaytuIoKaytuEnginePkgInventoryApiMetric, GithubComKaytuIoKaytuEnginePkgInventoryApiSpendTableRow } from '../../../../api/api'
import { badgeDelta } from '../../../../utilities/deltaType'
import { useInventoryApiV2AnalyticsMetricList } from '../../../../api/inventory.gen'
import { MSort } from '../../../Spend/Account/AccountTable'
import AdvancedTable from '../../../../components/AdvancedTable'
import { IFilter } from '../../../../utilities/urlstate'

interface IMetricTable {
    timeRange: { start: Dayjs; end: Dayjs }
    connections: IFilter
}

export const rowGenerator = (
    data: GithubComKaytuIoKaytuEnginePkgInventoryApiMetric[]
) => {
    const rows = []
    if (data) {
        for (let i = 0; i < data.length; i += 1) {
            if ((data[i].tags?.category.length || 0) > 1) {
                for (
                    let j = 0;
                    j < (data[i].tags?.category.length || 0);
                    j += 1
                ) {
                    rows.push({
                        ...data[i],
                        count: data[i].count || 0,
                        category: data[i].tags?.category[j],
                        change_percent:
                            (((data[i].old_count || 0) - (data[i].count || 0)) /
                                (data[i].count || 1)) *
                            100,
                        change_delta:
                            (data[i].old_count || 0) - (data[i].count || 0),
                    })
                }
            } else {
                rows.push({
                    ...data[i],
                    count: data[i].count || 0,
                    category: data[i].tags?.category[0],
                    change_percent:
                        (((data[i].old_count || 0) - (data[i].count || 0)) /
                            (data[i].count || 1)) *
                        100,
                    change_delta:
                        (data[i].old_count || 0) - (data[i].count || 0),
                })
            }
        }
    }

    return rows
}

export const SpendrowGenerator = (
    input:
        | GithubComKaytuIoKaytuEnginePkgInventoryApiSpendTableRow[]
        | undefined,
    inputPrev:
        | GithubComKaytuIoKaytuEnginePkgInventoryApiSpendTableRow[]
        | undefined,
    loading: boolean
) => {
    let sum = 0
    const roww = []
    const granularity: any = {}
    let pinnedRow = [
        { totalCost: sum, dimension: 'aTotal spend', ...granularity },
    ]
    if (!loading) {
        const rows =
            input?.map((row) => {
                let temp = {}
                let totalCost = 0
                if (row.costValue) {
                    temp = Object.fromEntries(Object.entries(row.costValue))
                }
                Object.values(temp).map(
                    // eslint-disable-next-line no-return-assign
                    (v: number | unknown) => (totalCost += Number(v))
                )
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
                    ...temp,
                }
            }) || []
        for (let i = 0; i < rows.length; i += 1) {
            sum += rows[i].totalCost
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
            { totalCost: sum, dimension: 'bTotal spend', ...granularity },
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


export const defaultColumns: IColumn<any, any>[] = [
    {
        headerName: 'Cloud provider',
        field: 'connectors',
        type: 'string',
        filter: true,
        width: 140,
        enableRowGroup: true,
    },
    {
        field: 'name',
        headerName: 'Metric',
        resizable: true,
        filter: true,
        sortable: true,
        type: 'string',
    },
    {
        field: 'category',
        headerName: 'Category',
        resizable: true,
        filter: true,
        sortable: true,
        type: 'string',
    },
    {
        field: 'count',
        resizable: true,
        sortable: true,
        headerName: 'Resource count',
        aggFunc: 'sum',
        filter: true,
        type: 'number',
    },
    {
        field: 'old_count',
        resizable: true,
        sortable: true,
        hide: true,
        headerName: 'Old resource count',
        aggFunc: 'sum',
        filter: true,
        type: 'number',
    },
    {
        headerName: 'Change (%)',
        field: 'change_percent',
        aggFunc: 'sum',
        sortable: true,
        type: 'number',
        filter: true,
        cellRenderer: (
            params: ICellRendererParams<GithubComKaytuIoKaytuEnginePkgInventoryApiMetric>
        ) =>
            params.data
                ? badgeDelta(params.data?.old_count, params.data?.count)
                : badgeDelta(params.value / 100, 0),
    },
    {
        headerName: 'Change (Δ)',
        field: 'change_delta',
        aggFunc: 'sum',
        sortable: true,
        type: 'number',
        hide: false,
        cellRenderer: (
            params: ICellRendererParams<GithubComKaytuIoKaytuEnginePkgInventoryApiMetric>
        ) =>
            params.data
                ? badgeDelta(params.data?.old_count, params.data?.count, true)
                : badgeDelta(params.value, 0, true),
    },
    {
        field: 'last_evaluated',
        resizable: true,
        sortable: true,
        headerName: 'Last evaluated',
        hide: true,
        filter: true,
        type: 'datetime',
    },
]

export const options: GridOptions = {
    enableGroupEdit: true,
    rowGroupPanelShow: 'always',
    groupAllowUnbalanced: true,
    autoGroupColumnDef: {
        width: 200,
        sortable: true,
        filter: true,
        resizable: true,
    },
}

export default function MetricTable({ timeRange, connections }: IMetricTable) {
    const navigate = useNavigate()
    const setNotification = useSetAtom(notificationAtom)
    const [searchParams, setSearchParams] = useSearchParams()

    const [manualGrouping, onManualGrouping] = useState<string>(
        searchParams.get('groupby') === 'category' ? 'category' : 'none'
    )
    const [manualTableSort, onManualSortChange] = useState<MSort>(
        searchParams.get('groupby')
            ? {
                  sortCol: 'none',
                  sortType: null,
              }
            : {
                  sortCol: 'count',
                  sortType: 'desc',
              }
    )
    const [tab, setTab] = useState(
        searchParams.get('groupby') === 'category' ? 2 : 0
    )
    const [tableKey, setTableKey] = useState('')

    const filterTabs = [
        {
            type: 0,
            icon: CircleStackIcon,
            name: 'Sort by Resource Count',
            function: () => {
                onManualSortChange({
                    sortCol: 'count',
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
                    sortCol: 'change_delta',
                    sortType: 'desc',
                })
                onManualGrouping('none')
            },
        },
        {
            type: 2,
            icon: Squares2X2Icon,
            name: 'Group by Category',
            function: () => {
                onManualGrouping('category')
                onManualSortChange({
                    sortCol: 'count',
                    sortType: 'desc',
                })
            },
        },
        {
            type: 3,
            icon: CloudIcon,
            name: 'Group by Provider',
            function: () => {
                onManualGrouping('connectors')
                onManualSortChange({
                    sortCol: 'count',
                    sortType: 'desc',
                })
            },
        },
    ]

    const query = {
        ...(connections.provider && {
            connector: [connections.provider],
        }),
        ...(connections.connections && {
            connectionId: connections.connections,
        }),
        ...(connections.connectionGroup && {
            connectionGroup: connections.connectionGroup,
        }),
        // ...(resourceId && {
        //     resourceCollection: [resourceId],
        // }),
        ...(timeRange.start && {
            startTime: timeRange.start.unix(),
        }),
        ...(timeRange.end && {
            endTime: timeRange.end.unix(),
        }),
        pageSize: 1000,
        needCost: false,
    }
    const { response: resources, isLoading: resourcesLoading } =
        useInventoryApiV2AnalyticsMetricList(query)

    useEffect(() => {
        setTableKey(Math.random().toString(16).slice(2, 8))
    }, [manualGrouping, timeRange, resources])

    return (
        <AdvancedTable
            key={`metric_${tableKey}`}
            id="asset_metric_table"
            title="Metric list"
            downloadable
            columns={defaultColumns}
            rowData={rowGenerator(resources?.metrics || []).sort((a, b) => {
                if ((a.category || '') < (b.category || '')) {
                    return -1
                }
                if ((a.category || '') > (b.category || '')) {
                    return 1
                }
                return 0
            })}
            tabIdx={tab}
            setTabIdx={setTab}
            manualSort={manualTableSort}
            manualGrouping={manualGrouping}
            filterTabs={filterTabs}
            loading={resourcesLoading}
            onRowClicked={(event) => {
                if (event.data) {
                    if (event.data.category) {
                        navigate(`metric_${event.data.id}?${searchParams}`)
                    } else {
                        setNotification({
                            text: 'Account is not onboarded',
                            type: 'warning',
                        })
                    }
                }
            }}
            onGridReady={(event) => {
                if (resourcesLoading) {
                    event.api.showLoadingOverlay()
                }
            }}
            options={options}
        />
    )
}
