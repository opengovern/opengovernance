import { Dayjs } from 'dayjs'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { useAtomValue, useSetAtom } from 'jotai/index'
import { useEffect, useState } from 'react'
import {
    ArrowTrendingUpIcon,
    CircleStackIcon,
    CloudIcon,
    ListBulletIcon,
} from '@heroicons/react/24/outline'
import { ICellRendererParams, ValueFormatterParams } from 'ag-grid-community'
import { isDemoAtom, notificationAtom } from '../../../../store'
import { IColumn } from '../../../../components/Table'
import {
    GithubComKaytuIoKaytuEnginePkgOnboardApiConnection,
    GithubComKaytuIoKaytuEngineServicesIntegrationApiEntityListConnectionsSummaryResponse,
} from '../../../../api/api'
import { badgeDelta } from '../../../../utilities/deltaType'
import { useIntegrationApiV1ConnectionsSummariesList } from '../../../../api/integration.gen'
import { MSort } from '../../../Spend/Account/AccountTable'
import AdvancedTable from '../../../../components/AdvancedTable'
import { options } from '../../Metric/Table'
import { IFilter, searchAtom } from '../../../../utilities/urlstate'

interface IAccountTable {
    timeRange: { start: Dayjs; end: Dayjs }
    connections: IFilter
}

export const cloudAccountColumns = (isDemo: boolean) => {
    const temp: IColumn<any, any>[] = [
        {
            field: 'connector',
            headerName: 'Cloud provider',
            type: 'string',
            width: 140,
            sortable: true,
            filter: true,
            enableRowGroup: true,
        },
        {
            field: 'providerConnectionName',
            headerName: 'Account name',
            resizable: true,
            type: 'string',
            sortable: true,
            filter: true,
            cellRenderer: (param: ValueFormatterParams) => (
                <span className={isDemo ? 'blur-sm' : ''}>{param.value}</span>
            ),
        },
        {
            field: 'providerConnectionID',
            headerName: 'Account ID',
            type: 'string',
            resizable: true,
            sortable: true,
            filter: true,
            cellRenderer: (param: ValueFormatterParams) => (
                <span className={isDemo ? 'blur-sm' : ''}>{param.value}</span>
            ),
        },
        {
            field: 'resourceCount',
            headerName: 'Resource count',
            type: 'number',
            aggFunc: 'sum',
            resizable: true,
            sortable: true,
        },
        {
            field: 'oldResourceCount',
            headerName: 'Old resource count',
            type: 'number',
            aggFunc: 'sum',
            hide: true,
            resizable: true,
            sortable: true,
        },
        {
            headerName: 'Change (%)',
            field: 'change_percent',
            aggFunc: 'sum',
            sortable: true,
            type: 'number',
            filter: true,
            cellRenderer: (
                params: ICellRendererParams<GithubComKaytuIoKaytuEnginePkgOnboardApiConnection>
            ) =>
                params.data
                    ? badgeDelta(
                          params.data?.oldResourceCount,
                          params.data?.resourceCount
                      )
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
                params: ICellRendererParams<GithubComKaytuIoKaytuEnginePkgOnboardApiConnection>
            ) =>
                params.data
                    ? badgeDelta(
                          params.data?.oldResourceCount,
                          params.data?.resourceCount,
                          true
                      )
                    : badgeDelta(params.value, 0, true),
        },
        {
            field: 'lastInventory',
            headerName: 'Last inventory',
            type: 'datetime',
            hide: true,
            resizable: true,
            sortable: true,
        },
        {
            field: 'onboardDate',
            headerName: 'Onboard Date',
            type: 'datetime',
            hide: true,
            resizable: true,
            sortable: true,
        },
    ]
    return temp
}

const rowGenerator = (
    data:
        | GithubComKaytuIoKaytuEngineServicesIntegrationApiEntityListConnectionsSummaryResponse
        | undefined
) => {
    const rows = []
    if (data && data.connections) {
        for (let i = 0; i < data.connections.length; i += 1) {
            rows.push({
                ...data.connections[i],
                resourceCount: data.connections[i].resourceCount || 0,
                change_percent:
                    (((data.connections[i].oldResourceCount || 0) -
                        (data.connections[i].resourceCount || 0)) /
                        (data.connections[i].resourceCount || 1)) *
                    100,
                change_delta:
                    (data.connections[i].oldResourceCount || 0) -
                    (data.connections[i].resourceCount || 0),
            })
        }
    }
    return rows
}

export default function AccountTable({
    timeRange,
    connections,
}: IAccountTable) {
    const searchParams = useAtomValue(searchAtom)
    const navigate = useNavigate()
    const setNotification = useSetAtom(notificationAtom)
    const isDemo = useAtomValue(isDemoAtom)

    const [manualGrouping, onManualGrouping] = useState<string>('none')
    const [manualTableSort, onManualSortChange] = useState<MSort>({
        sortCol: 'resourceCount',
        sortType: 'desc',
    })
    const [tab, setTab] = useState(0)
    const [tableKey, setTableKey] = useState('')

    const filterTabs = [
        {
            type: 0,
            icon: CircleStackIcon,
            name: 'Sort by Resource Count',
            function: () => {
                onManualSortChange({
                    sortCol: 'resourceCount',
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
            icon: ArrowTrendingUpIcon,
            name: 'Sort by Account Name',
            function: () => {
                onManualSortChange({
                    sortCol: 'providerConnectionName',
                    sortType: 'asc',
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
                    sortCol: 'resourceCount',
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
        pageSize: 5000,
        needCost: false,
    }

    const { response: accounts, isLoading: isAccountsLoading } =
        useIntegrationApiV1ConnectionsSummariesList(query)

    useEffect(() => {
        setTableKey(Math.random().toString(16).slice(2, 8))
    }, [manualGrouping, timeRange, accounts])

    return (
        <AdvancedTable
            key={`account_${tableKey}`}
            id="asset_connection_table"
            title="Cloud account list"
            downloadable
            columns={cloudAccountColumns(isDemo)}
            rowData={rowGenerator(accounts)
                ?.sort(
                    (a, b) => (b.resourceCount || 0) - (a.resourceCount || 0)
                )
                .filter((acc) => {
                    return (
                        acc.lifecycleState === 'ONBOARD' ||
                        acc.lifecycleState === 'IN_PROGRESS'
                    )
                })}
            tabIdx={tab}
            setTabIdx={setTab}
            manualSort={manualTableSort}
            manualGrouping={manualGrouping}
            filterTabs={filterTabs}
            onRowClicked={(event) => {
                if (event.data.id) {
                    if (event.data.lifecycleState === 'ONBOARD') {
                        navigate(`account_${event.data.id}?${searchParams}`)
                    } else {
                        setNotification({
                            text: 'Account is not onboarded',
                            type: 'warning',
                        })
                    }
                }
            }}
            loading={isAccountsLoading}
            onGridReady={(event) => {
                if (isAccountsLoading) {
                    event.api.showLoadingOverlay()
                }
            }}
            options={options}
        />
    )
}
