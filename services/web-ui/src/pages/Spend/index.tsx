import { dateDisplay, monthDisplay } from '../../utilities/dateDisplay'
import {
    GithubComKaytuIoKaytuEnginePkgInventoryApiCostStackedItem,
    GithubComKaytuIoKaytuEnginePkgInventoryApiCostTrendDatapoint,
    GithubComKaytuIoKaytuEnginePkgInventoryApiListCostCompositionResponse,
    GithubComKaytuIoKaytuEnginePkgInventoryApiListCostMetricsResponse,
    GithubComKaytuIoKaytuEngineServicesIntegrationApiEntityListConnectionsSummaryResponse,
    SourceType,
} from '../../api/api'
import { StackItem } from '../../components/Chart/Stacked'
import { SpendOverview } from './Overview'

export const topServices = (
    input:
        | GithubComKaytuIoKaytuEnginePkgInventoryApiListCostMetricsResponse
        | undefined,
    addRateChange: boolean
) => {
    const top: {
        data: {
            name: string | undefined
            value: number | undefined
            valueRateChange?: number | undefined
            connector: SourceType[] | undefined
            kaytuId: string | undefined
        }[]
        total: number | undefined
    } = { data: [], total: 0 }
    if (input && input.metrics) {
        for (let i = 0; i < input.metrics.length; i += 1) {
            top.data.push({
                name: input.metrics[i].cost_dimension_name,
                value: input.metrics[i].total_cost,
                valueRateChange: addRateChange
                    ? (((input.metrics[i].daily_cost_at_end_time || 0) -
                          (input.metrics[i].daily_cost_at_start_time || 0)) /
                          (input.metrics[i].daily_cost_at_start_time || 1)) *
                      100.0
                    : undefined,
                connector: input.metrics[i].connector,
                kaytuId: input.metrics[i].cost_dimension_id,
            })
        }
        top.total = input.total_count
    }
    return top
}

export const topAccounts = (
    input:
        | GithubComKaytuIoKaytuEngineServicesIntegrationApiEntityListConnectionsSummaryResponse
        | undefined,
    addRateChange: boolean
) => {
    const top: {
        data: {
            name: string | undefined
            value: number | undefined
            valueRateChange?: number | undefined
            connector: SourceType[]
            id: string | undefined
            kaytuId: string | undefined
        }[]
        total: number | undefined
    } = { data: [], total: 0 }
    if (input && input.connections) {
        for (let i = 0; i < input.connections.length; i += 1) {
            top.data.push({
                name: input.connections[i].providerConnectionName,
                value: input.connections[i].cost,
                valueRateChange: addRateChange
                    ? (((input.connections[i].dailyCostAtEndTime || 0) -
                          (input.connections[i].dailyCostAtStartTime || 0)) /
                          (input.connections[i].dailyCostAtStartTime || 1)) *
                      100.0
                    : undefined,
                connector: [input.connections[i].connector || SourceType.Nil],
                id: input.connections[i].providerConnectionID,
                kaytuId: input.connections[i].id,
            })
        }
        top.total = input.connectionCount
    }
    return top
}

export const topCategories = (
    input:
        | GithubComKaytuIoKaytuEnginePkgInventoryApiListCostCompositionResponse
        | undefined
) => {
    const top: {
        data: {
            name: string | undefined
            value: number | undefined
            connector: SourceType[]
        }[]
        total: number | undefined
    } = { data: [], total: 0 }
    if (input && input.top_values) {
        const arr = Object.entries(input.top_values)
        for (let i = 0; i < arr.length; i += 1) {
            top.data.push({
                name: arr[i][0],
                value: arr[i][1],
                connector: [SourceType.CloudAWS, SourceType.CloudAzure],
            })
        }
        top.data.sort((a, b) => {
            if (a.value === b.value) {
                return 0
            }
            return (a.value || 0) < (b.value || 0) ? 1 : -1
        })
        top.total = input.total_count
    }
    return top
}

const topFiveStackedMetrics = (
    data: GithubComKaytuIoKaytuEnginePkgInventoryApiCostTrendDatapoint[]
) => {
    const uniqueMetricID = data
        .flatMap((v) => v.costStacked?.map((i) => i.metricID || '') || [])
        .filter((l, idx, arr) => arr.indexOf(l) === idx)

    const idCost = uniqueMetricID
        .map((metricID) => {
            const totalCost = data
                .flatMap(
                    (v) =>
                        v.costStacked
                            ?.filter((i) => i.metricID === metricID)
                            .map((j) => j.cost || 0) || []
                )
                .reduce((prev, curr) => prev + curr, 0)

            const metricName =
                data
                    .flatMap(
                        (v) =>
                            v.costStacked
                                ?.filter((i) => i.metricID === metricID)
                                .map((j) => j.metricName || '') || []
                    )
                    .at(0) || ''

            return {
                metricID,
                metricName,
                totalCost,
            }
        })
        .sort((a, b) => {
            if (a.totalCost === b.totalCost) {
                return 0
            }
            return a.totalCost < b.totalCost ? 1 : -1
        })

    return idCost.slice(0, 5)
}

const takeMetricsAndOthers = (
    metricIDs: {
        metricID: string
        metricName: string
        totalCost: number
    }[],
    v: GithubComKaytuIoKaytuEnginePkgInventoryApiCostStackedItem[]
) => {
    const result: GithubComKaytuIoKaytuEnginePkgInventoryApiCostStackedItem[] =
        []
    let others = 0
    v.forEach((item) => {
        if (
            metricIDs.map((i) => i.metricID).indexOf(item.metricID || '') === -1
        ) {
            others += item.cost || 0
        }
    })

    metricIDs.forEach((item) => {
        const p = v.filter((i) => i.metricID === item.metricID).at(0)
        if (p === undefined) {
            result.push({
                metricID: item.metricID,
                metricName: item.metricName,
                cost: 0,
            })
        } else {
            result.push(p)
        }
    })

    result.push({
        metricID: '___others___',
        metricName: 'Others',
        cost: others,
    })

    return result
}

export const costTrendChart = (
    trend:
        | GithubComKaytuIoKaytuEnginePkgInventoryApiCostTrendDatapoint[]
        | undefined,
    chart: 'trend' | 'cumulative',
    layout: 'basic' | 'stacked',
    granularity: 'monthly' | 'daily' | 'yearly'
) => {
    const top5 = topFiveStackedMetrics(trend || [])
    const label = []
    const data: any = []
    const flag = []
    if (trend) {
        if (chart === 'trend') {
            for (let i = 0; i < trend?.length; i += 1) {
                const stacked = takeMetricsAndOthers(
                    top5,
                    trend[i].costStacked || []
                )
                label.push(
                    granularity === 'monthly'
                        ? monthDisplay(trend[i]?.date)
                        : dateDisplay(trend[i]?.date)
                )
                if (layout === 'basic') {
                    data.push(trend[i]?.cost)
                } else {
                    data.push(
                        stacked.map((v) => {
                            const j: StackItem = {
                                label: v.metricName || '',
                                value: v.cost || 0,
                            }
                            return j
                        })
                    )
                }
                if (
                    trend[i].totalConnectionCount !==
                    trend[i].totalSuccessfulDescribedConnectionCount
                ) {
                    flag.push(true)
                } else flag.push(false)
            }
        }
        if (chart === 'cumulative') {
            for (let i = 0; i < trend?.length; i += 1) {
                const stacked = takeMetricsAndOthers(
                    top5,
                    trend[i].costStacked || []
                )
                label.push(
                    granularity === 'monthly'
                        ? monthDisplay(trend[i]?.date)
                        : dateDisplay(trend[i]?.date)
                )

                if (i === 0) {
                    if (layout === 'basic') {
                        data.push(trend[i]?.cost)
                    } else {
                        data.push(
                            stacked.map((v) => {
                                const j: StackItem = {
                                    label: v.metricName || '',
                                    value: v.cost || 0,
                                }
                                return j
                            })
                        )
                    }
                } else if (layout === 'basic') {
                    data.push((trend[i]?.cost || 0) + data[i - 1])
                } else {
                    data.push(
                        stacked.map((v) => {
                            const prev = data[i - 1]
                                ?.filter((p: any) => p.label === v.metricName)
                                ?.at(0)

                            const j: StackItem = {
                                label: v.metricName || '',
                                value: (v.cost || 0) + (prev?.value || 0),
                            }
                            return j
                        })
                    )
                }
            }
        }
    }
    return {
        label,
        data,
        flag,
    }
}

export const pieData = (
    response:
        | GithubComKaytuIoKaytuEnginePkgInventoryApiListCostCompositionResponse
        | undefined
) => {
    const data: any[] = []
    if (response && response.top_values) {
        Object.entries(response?.top_values).map(([key, value]) =>
            data.push({
                name: key,
                value: Number(value).toFixed(0),
            })
        )
        data.sort((a, b) => {
            return b.value - a.value
        })
        data.push({
            name: 'Others',
            value: Number(response.others).toFixed(0),
        })
    }
    return data
}

export default function Spend() {
    return <SpendOverview />
}
