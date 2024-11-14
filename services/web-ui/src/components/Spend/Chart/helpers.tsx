import { GithubComKaytuIoKaytuEnginePkgInventoryApiCostTrendDatapoint } from '../../../api/api'
import { dateDisplay, monthDisplay } from '../../../utilities/dateDisplay'
import { StackItem } from '../../Chart/Stacked'

export const costTrendChart = (
    trend:
        | GithubComKaytuIoKaytuEnginePkgInventoryApiCostTrendDatapoint[]
        | undefined,
    chart: 'trending' | 'aggregated',
    granularity: 'monthly' | 'daily' | 'yearly'
) => {
    const label: string[] = []
    const data: number[] = []
    const flag: boolean[] = []

    if (!trend) {
        return {
            label,
            data,
            flag,
        }
    }

    for (let i = 0; i < trend?.length; i += 1) {
        label.push(
            granularity === 'monthly'
                ? monthDisplay(trend[i]?.date)
                : dateDisplay(trend[i]?.date)
        )
        if (chart === 'aggregated') {
            data.push((trend[i]?.cost || 0) + (data?.at(i - 1) || 0))
        } else {
            data.push(trend[i]?.cost || 0)
        }

        if (
            trend[i].totalConnectionCount !==
            trend[i].totalSuccessfulDescribedConnectionCount
        ) {
            flag.push(true)
        } else flag.push(false)
    }

    return {
        label,
        data,
        flag,
    }
}

interface ITrendItem {
    date: string
    totalValue: number
    stackedValues: StackItem[]
    flag: boolean
}

const makeUnique = (arr: StackItem[]) => {
    return arr.reduce<StackItem[]>((prev, curr) => {
        const exists = prev.filter((v) => v.label === curr.label).length > 0
        if (exists) {
            return prev.map((v) =>
                v.label === curr.label
                    ? {
                          label: v.label,
                          value: v.value + curr.value,
                      }
                    : v
            )
        }
        return [...prev, curr]
    }, [])
}

const extractTrend = (
    trend: GithubComKaytuIoKaytuEnginePkgInventoryApiCostTrendDatapoint[],
    granularity: 'monthly' | 'daily' | 'yearly'
) => {
    return trend?.map((item) => {
        const label =
            granularity === 'monthly'
                ? monthDisplay(item?.date)
                : dateDisplay(item?.date)

        const dataItem: StackItem[] =
            item.costStacked?.flatMap((v) => {
                const labels = [v.metricName || '']
                return labels.map((lbl) => {
                    return {
                        label: lbl,
                        value: v.cost || 0,
                    }
                })
            }) || []

        const i: ITrendItem = {
            date: label,
            totalValue: item.cost || 0,
            stackedValues: makeUnique(dataItem),
            flag:
                item.totalConnectionCount !==
                item.totalSuccessfulDescribedConnectionCount,
        }
        return i
    })
}

export const buildTrend = (
    apiResp: GithubComKaytuIoKaytuEnginePkgInventoryApiCostTrendDatapoint[],
    chart: 'trending' | 'aggregated',
    granularity: 'monthly' | 'daily' | 'yearly',
    topN: number
) => {
    let trend = extractTrend(apiResp, granularity)

    const order = makeUnique(trend.flatMap((v) => v.stackedValues)).sort(
        (a, b) => {
            if (a.value === b.value) {
                return 0
            }
            return a.value < b.value ? 1 : -1
        }
    )
    const topNLabels = order.slice(0, topN).map((v) => v.label)

    const orderMap = new Map(order.map((obj) => [obj.label, obj.value]))

    trend = trend?.map((v) => {
        return {
            ...v,
            stackedValues: makeUnique(
                v.stackedValues.map((s) => ({
                    label:
                        topNLabels.filter((l) => l === s.label).length > 0
                            ? s.label
                            : 'Others',
                    value: s.value,
                }))
            ),
        }
    })

    if (chart === 'aggregated') {
        trend = trend.reduce<ITrendItem[]>((prev, curr) => {
            const p = prev.at(prev.length - 1)
            const c = {
                ...curr,
                totalValue: (p?.totalValue || 0) + curr.totalValue,
                stackedValues: curr.stackedValues.map((s) => {
                    return {
                        label: s.label,
                        value:
                            (p?.stackedValues
                                .filter((i) => i.label === s.label)
                                .at(0)?.value || 0) + s.value,
                    }
                }),
            }
            return [...prev, c]
        }, [])
    }

    trend = trend?.map((v) => {
        return {
            ...v,
            stackedValues: v.stackedValues.sort((a, b) => {
                const av = orderMap.get(a.label) || 0
                const bv = orderMap.get(b.label) || 0
                if (av === bv) {
                    return 0
                }
                return av < bv ? 1 : -1
            }),
        }
    })

    return {
        label: trend?.map((v) => v.date),
        data: trend?.map((v) => v.stackedValues),
        flag: trend?.map((v) => v.flag),
    }
}
