import { Callout, Card, Col, Grid } from '@tremor/react'
import dayjs from 'dayjs'
import { useEffect, useState } from 'react'
import { numberDisplay } from '../../../utilities/numericDisplay'
import StackedChart from '../../Chart/Stacked'
import Chart from '../../Chart'
import { dateDisplay } from '../../../utilities/dateDisplay'
import { SpendChartMetric } from './Metric'
import {
    Aggregation,
    ChartLayout,
    ChartType,
    Granularity,
    SpendChartSelectors,
} from './Selectors'
import { buildTrend, costTrendChart } from './helpers'
import { generateVisualMap } from '../../../pages/Assets'
import { GithubComKaytuIoKaytuEnginePkgInventoryApiCostTrendDatapoint } from '../../../api/api'
import { errorHandlingWithErrorMessage } from '../../../types/apierror'
import { useURLParam } from '../../../utilities/urlstate'

interface ISpendChart {
    title: string
    timeRange: {
        start: dayjs.Dayjs
        end: dayjs.Dayjs
    }
    total: number
    timeRangePrev: {
        start: dayjs.Dayjs
        end: dayjs.Dayjs
    }
    totalPrev: number
    isLoading: boolean
    costTrend: GithubComKaytuIoKaytuEnginePkgInventoryApiCostTrendDatapoint[]
    error: string | undefined
    onRefresh: () => void
    onGranularityChanged: (v: Granularity) => void
    noStackedChart?: boolean
    chartLayout: ChartLayout
    setChartLayout: (v: ChartLayout) => void
    validChartLayouts: ChartLayout[]
}

export function SpendChart({
    title,
    timeRange,
    timeRangePrev,
    total,
    totalPrev,
    costTrend,
    noStackedChart,
    onGranularityChanged,
    isLoading,
    error,
    onRefresh,
    chartLayout,
    setChartLayout,
    validChartLayouts,
}: ISpendChart) {
    const [selectedDatapoint, setSelectedDatapoint] = useState<any>(undefined)
    const [chartType, setChartType] = useURLParam<ChartType>('chartType', 'bar')
    const [granularity, setGranularity] = useURLParam<Granularity>(
        'granularity',
        'daily'
    )
    const [aggregation, setAggregation] = useURLParam<Aggregation>(
        'aggregation',
        'trending'
    )

    const trend = costTrendChart(costTrend, aggregation, granularity)
    const trendStacked = buildTrend(costTrend, aggregation, granularity, 5)
    const visualMap = generateVisualMap(trend.flag, trend.label)
    useEffect(() => {
        setSelectedDatapoint(undefined)
    }, [chartLayout, chartType, granularity, aggregation])

    return (
        <Card>
            <Grid numItems={6} className="gap-4 mb-4">
                <Col numColSpan={2}>
                    <SpendChartMetric
                        title={title}
                        timeRange={timeRange}
                        total={total}
                        timeRangePrev={timeRangePrev}
                        totalPrev={totalPrev}
                        isLoading={isLoading}
                        error={error}
                    />
                </Col>
                <Col numColSpan={4}>
                    <SpendChartSelectors
                        timeRange={timeRange}
                        chartType={chartType}
                        setChartType={setChartType}
                        granularity={granularity}
                        setGranularity={(v) => {
                            setGranularity(v)
                            onGranularityChanged(v)
                        }}
                        chartLayout={chartLayout}
                        setChartLayout={setChartLayout}
                        validChartLayouts={validChartLayouts}
                        aggregation={aggregation}
                        setAggregation={setAggregation}
                        noStackedChart={noStackedChart}
                    />
                </Col>
            </Grid>
            {chartLayout === 'total' && selectedDatapoint !== undefined && (
                <Callout
                    color="rose"
                    title="Incomplete data"
                    className="w-fit mt-4"
                >
                    Checked{' '}
                    {numberDisplay(
                        selectedDatapoint.totalSuccessfulDescribedConnectionCount,
                        0
                    )}{' '}
                    accounts out of{' '}
                    {numberDisplay(selectedDatapoint.totalConnectionCount, 0)}{' '}
                    on {dateDisplay(selectedDatapoint.date)}
                </Callout>
            )}

            {chartLayout !== 'total' ? (
                <StackedChart
                    labels={trendStacked.label}
                    chartData={trendStacked.data}
                    isCost
                    chartType={chartType}
                    loading={isLoading}
                    error={error}
                />
            ) : (
                <Chart
                    labels={trend.label}
                    chartData={trend.data}
                    chartType={chartType}
                    chartAggregation={
                        aggregation === 'trending' ? 'trend' : 'cumulative'
                    }
                    isCost
                    loading={isLoading}
                    error={error}
                    visualMap={
                        aggregation === 'aggregated'
                            ? undefined
                            : visualMap.visualMap
                    }
                    markArea={
                        aggregation === 'aggregated'
                            ? undefined
                            : visualMap.markArea
                    }
                    onClick={(p) => {
                        if (aggregation !== 'aggregated') {
                            const t1 = costTrend
                                ?.filter(
                                    (t) =>
                                        p?.color === '#E01D48' &&
                                        dateDisplay(t.date) === p?.name
                                )
                                .at(0)

                            setSelectedDatapoint(t1)
                        }
                    }}
                />
            )}

            {errorHandlingWithErrorMessage(onRefresh, error)}
        </Card>
    )
}
