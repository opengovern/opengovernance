import { Dayjs } from 'dayjs'
import { GridOptions, ValueFormatterParams } from 'ag-grid-community'
import { Dispatch, SetStateAction } from 'react'
import { IColumn } from '../../../../../components/Table'
import { GithubComKaytuIoKaytuEnginePkgInventoryApiSpendTableRow } from '../../../../../api/api'
import { exactPriceDisplay } from '../../../../../utilities/numericDisplay'
import { IFilter } from '../../../../../utilities/urlstate'

type MSort = {
    sortCol: string
    sortType: 'asc' | 'desc' | null
}

interface IServices {
    activeTimeRange: { start: Dayjs; end: Dayjs }
    connections: IFilter
    selectedGranularity: 'monthly' | 'daily'
    isSummary?: boolean
    onGranularityChange: Dispatch<SetStateAction<'monthly' | 'daily'>>
}

export const rowGenerator = (
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
        field: 'connector',
        headerName: 'Cloud provider',
        type: 'string',
        width: 140,
        enableRowGroup: true,
        filter: true,
        resizable: true,
        sortable: true,
        pinned: true,
    },
    {
        field: 'dimension',
        headerName: 'Metric',
        type: 'string',
        filter: true,
        sortable: true,
        resizable: true,
        pivot: false,
        pinned: true,
    },
    {
        field: 'totalCost',
        headerName: 'Total spend',
        type: 'price',
        width: 110,
        sortable: true,
        aggFunc: 'sum',
        resizable: true,
        pivot: false,
        pinned: true,
        valueFormatter: (param: ValueFormatterParams) => {
            return param.value ? exactPriceDisplay(param.value) : ''
        },
    },
]

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
        flex: 2,
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
    groupIncludeFooter: true,
    groupIncludeTotalFooter: true,
}
