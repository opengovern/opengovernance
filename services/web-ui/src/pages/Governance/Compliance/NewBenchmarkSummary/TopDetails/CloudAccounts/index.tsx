import { GridOptions, ValueFormatterParams } from 'ag-grid-community'
import { useAtomValue } from 'jotai'
import { useComplianceApiV1FindingsAccountsDetail } from '../../../../../../api/compliance.gen'
import Table, { IColumn } from '../../../../../../components/Table'
import { dateTimeDisplay } from '../../../../../../utilities/dateDisplay'
import { isDemoAtom } from '../../../../../../store'
import { IFilter } from '../../../../../../utilities/urlstate'

interface IFinder {
    id: string | undefined
    connections: IFilter
    resourceId: string | undefined
}

const rowGenerator = (data: any) => {
    const temp: any = []
    if (data && data?.accounts) {
        const holder = data?.accounts
        for (let i = 0; i < holder.length; i += 1) {
            temp.push({ ...holder[i], ...holder[i].SeveritiesCount })
        }
    }
    return temp
}

const columns = (isDemo: boolean) => {
    const temp: IColumn<any, any>[] = [
        {
            field: 'accountName',
            headerName: 'Account name',
            type: 'string',
            sortable: true,
            filter: true,
            resizable: true,
            flex: 1,
            cellRenderer: (param: ValueFormatterParams) => (
                <span className={isDemo ? 'blur-sm' : ''}>{param.value}</span>
            ),
        },
        {
            field: 'accountId',
            headerName: 'Account ID',
            type: 'string',
            sortable: true,
            filter: true,
            resizable: true,
            flex: 1,
            cellRenderer: (param: ValueFormatterParams) => (
                <span className={isDemo ? 'blur-sm' : ''}>{param.value}</span>
            ),
        },
        {
            field: 'severitiesCount.critical',
            headerName: 'Critical',
            type: 'number',
            sortable: true,
            filter: true,
            resizable: true,
            flex: 0.5,
        },
        {
            field: 'severitiesCount.high',
            headerName: 'High',
            type: 'number',
            sortable: true,
            filter: true,
            resizable: true,
            flex: 0.5,
        },
        {
            field: 'severitiesCount.medium',
            headerName: 'Medium',
            type: 'number',
            sortable: true,
            filter: true,
            resizable: true,
            flex: 0.5,
        },
        {
            field: 'severitiesCount.low',
            headerName: 'Low',
            type: 'number',
            sortable: true,
            filter: true,
            resizable: true,
            flex: 0.5,
        },
        {
            field: 'securityScore',
            headerName: 'Security score',
            type: 'string',
            sortable: true,
            filter: true,
            resizable: true,
            flex: 0.5,
            valueFormatter: (param: ValueFormatterParams) => {
                return `${param.value ? Number(param.value).toFixed(2) : '0'}%`
            },
        },
        {
            field: 'lastCheckTime',
            headerName: 'Last checked',
            type: 'datetime',
            sortable: true,
            filter: true,
            resizable: true,
            flex: 1,
            hide: true,
            valueFormatter: (param: ValueFormatterParams) => {
                return param.value ? dateTimeDisplay(param.value) : ''
            },
        },
    ]
    return temp
}

export default function CloudAccounts({
    id,
    connections,
    resourceId,
}: IFinder) {
    const isDemo = useAtomValue(isDemoAtom)

    const { response: findings, isLoading } =
        useComplianceApiV1FindingsAccountsDetail(id || '', {
            connectionId: connections.connections,
            connectionGroup: connections.connectionGroup,
            ...(resourceId && {
                resourceCollection: [resourceId],
            }),
        })

    const options: GridOptions = {
        enableGroupEdit: true,
        columnTypes: {
            dimension: {
                enableRowGroup: true,
                enablePivot: true,
            },
        },
        rowGroupPanelShow: 'always',
        groupAllowUnbalanced: true,
    }

    return (
        <Table
            title="Cloud accounts"
            downloadable
            id="compliance_connections"
            columns={columns(isDemo)}
            rowData={rowGenerator(findings) || []}
            options={options}
            onGridReady={(e) => {
                if (isLoading) {
                    e.api.showLoadingOverlay()
                }
            }}
            loading={isLoading}
        />
    )
}
