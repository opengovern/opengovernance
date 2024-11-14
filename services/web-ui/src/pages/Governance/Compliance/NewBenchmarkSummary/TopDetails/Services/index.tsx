import { GridOptions, ValueFormatterParams } from 'ag-grid-community'
import { useComplianceApiV1FindingsServicesDetail } from '../../../../../../api/compliance.gen'
import Table, { IColumn } from '../../../../../../components/Table'
import { dateTimeDisplay } from '../../../../../../utilities/dateDisplay'
import { IFilter } from '../../../../../../utilities/urlstate'

interface IFinder {
    id: string | undefined
    connections: IFilter
    resourceId: string | undefined
}

const rowGenerator = (data: any) => {
    const temp: any = []
    if (data && data?.services) {
        const holder = data?.services
        for (let i = 0; i < holder.length; i += 1) {
            temp.push({ ...holder[i], ...holder[i].SeveritiesCount })
        }
    }
    return temp
}

const columns: IColumn<any, any>[] = [
    {
        field: 'serviceName',
        headerName: 'Resource name',
        type: 'string',
        sortable: true,
        filter: true,
        resizable: true,
        flex: 1,
    },
    {
        field: 'serviceLabel',
        headerName: 'Resource label',
        type: 'string',
        sortable: true,
        filter: true,
        resizable: true,
        flex: 1,
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

export default function Services({ id, connections, resourceId }: IFinder) {
    const { response: findings, isLoading } =
        useComplianceApiV1FindingsServicesDetail(id || '', {
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
            title="Resource types"
            downloadable
            id="compliance_services"
            columns={columns}
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
