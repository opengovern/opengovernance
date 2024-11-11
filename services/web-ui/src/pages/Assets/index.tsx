import {
    GithubComKaytuIoKaytuEnginePkgInventoryApiListMetricsResponse,
    GithubComKaytuIoKaytuEnginePkgInventoryApiListResourceTypeCompositionResponse,
    GithubComKaytuIoKaytuEnginePkgInventoryApiResourceTypeTrendDatapoint,
    GithubComKaytuIoKaytuEngineServicesIntegrationApiEntityListConnectionsSummaryResponse,
    SourceType,
} from '../../api/api'
import { dateDisplay, monthDisplay } from '../../utilities/dateDisplay'
import { AssetOverview } from './Overview'

export const resourceTrendChart = (
    trend:
        | GithubComKaytuIoKaytuEnginePkgInventoryApiResourceTypeTrendDatapoint[]
        | undefined,
    granularity: 'monthly' | 'daily' | 'yearly'
) => {
    const label = []
    const data: any = []
    const flag = []
    if (trend) {
        for (let i = 0; i < trend?.length; i += 1) {
            label.push(
                granularity === 'monthly'
                    ? monthDisplay(trend[i]?.date)
                    : dateDisplay(trend[i]?.date)
            )
            data.push(trend[i]?.count)
            if (
                trend[i].totalConnectionCount !==
                trend[i].totalSuccessfulDescribedConnectionCount
            ) {
                flag.push(true)
            } else flag.push(false)
        }
    }
    return {
        label,
        data,
        flag,
    }
}

export const generateVisualMap = (flag: boolean[], label: string[]) => {
    const pieces = []
    const data = []
    if (flag) {
        for (let i = 0; i < flag.length; i += 1) {
            pieces.push({
                gt: i - 1,
                lte: i,
                color: flag[i] ? '#E01D48' : '#1D4F85',
            })
        }
        for (let i = 0; i < pieces.length; i += 1) {
            if (pieces[i].color === '#E01D48') {
                data.push([
                    { xAxis: label[pieces[i].gt < 0 ? 0 : pieces[i].gt] },
                    { xAxis: label[pieces[i].lte] },
                ])
            }
        }
    }
    return {
        visualMap: pieces.length
            ? { show: false, dimension: 0, pieces }
            : undefined,
        markArea: data.length
            ? {
                  itemStyle: {
                      color: 'rgba(255, 173, 177, 0.1)',
                  },
                  data,
              }
            : undefined,
    }
}

export const pieData = (
    response:
        | GithubComKaytuIoKaytuEnginePkgInventoryApiListResourceTypeCompositionResponse
        | undefined
) => {
    const newData: any[] = []
    const oldData: any[] = []

    if (response && response.top_values) {
        // eslint-disable-next-line array-callback-return
        Object.entries(response?.top_values).map(([key, value]) => {
            newData.push({
                name: key,
                value: Number(value.count).toFixed(0),
            })
            oldData.push({
                name: key,
                value: Number(value.old_count).toFixed(0),
            })
        })
        newData.sort((a, b) => {
            return b.value - a.value
        })
        oldData.sort((a, b) => {
            return b.value - a.value
        })
        newData.push({
            name: 'Others',
            value: Number(response.others?.count).toFixed(0),
        })
        oldData.push({
            name: 'Others',
            value: Number(response.others?.old_count).toFixed(0),
        })
    }
    return { newData, oldData }
}

export const topAccounts = (
    input:
        | GithubComKaytuIoKaytuEngineServicesIntegrationApiEntityListConnectionsSummaryResponse
        | undefined
) => {
    const top: {
        data: {
            name: string | undefined
            value: number | undefined
            connector: SourceType[]
            id: string | undefined
            kaytuId: string | undefined
        }[]
        total: number | undefined
    } = { data: [], total: 0 }
    if (input && input.connections) {
        for (let i = 0; i < input.connections.length; i += 1) {
            top.data.push({
                kaytuId: input.connections[i].id,
                name: input.connections[i].providerConnectionName,
                value: input.connections[i].resourceCount,
                connector: [input.connections[i].connector || SourceType.Nil],
                id: input.connections[i].providerConnectionID,
            })
        }
        top.total = input.totalOnboardedCount
    }
    return top
}

export const topServices = (
    input:
        | GithubComKaytuIoKaytuEnginePkgInventoryApiListMetricsResponse
        | undefined
) => {
    const top: {
        data: {
            name: string | undefined
            value: number | undefined
            connector: SourceType[] | undefined
            kaytuId: string | undefined
        }[]
        total: number | undefined
    } = { data: [], total: 0 }
    if (input && input.metrics) {
        for (let i = 0; i < input.metrics.length; i += 1) {
            top.data.push({
                name: input.metrics[i].name,
                value: input.metrics[i].count,
                connector: input.metrics[i]?.connectors,
                kaytuId: input.metrics[i]?.id,
            })
        }
        top.total = input.total_metrics
    }
    return top
}

export default function Assets() {
    return <AssetOverview />
}
