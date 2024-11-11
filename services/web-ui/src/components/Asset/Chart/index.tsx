import { Callout, Card, Col, Grid } from '@tremor/react'
import dayjs from 'dayjs'
import { useEffect, useState } from 'react'
import { numberDisplay } from '../../../utilities/numericDisplay'
import StackedChart from '../../Chart/Stacked'
import Chart from '../../Chart'
import { dateDisplay } from '../../../utilities/dateDisplay'
import { AssetChartMetric } from './Metric'
import {
    Aggregation,
    AssetChartSelectors,
    ChartLayout,
    ChartType,
    Granularity,
} from './Selectors'
import { buildTrend, trendChart } from './helpers'
import { generateVisualMap } from '../../../pages/Assets'
import { GithubComKaytuIoKaytuEnginePkgInventoryApiResourceTypeTrendDatapoint } from '../../../api/api'
import { errorHandlingWithErrorMessage } from '../../../types/apierror'
import { useURLParam, useURLState } from '../../../utilities/urlstate'

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
    trend: GithubComKaytuIoKaytuEnginePkgInventoryApiResourceTypeTrendDatapoint[]
    error: string | undefined
    onRefresh: () => void
    onGranularityChanged: (v: Granularity) => void
    noStackedChart?: boolean
    chartLayout: ChartLayout
    setChartLayout: (v: ChartLayout) => void
    validChartLayouts: ChartLayout[]
}

export function AssetChart({
    title,
    timeRange,
    timeRangePrev,
    total,
    totalPrev,
    trend,
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

    const theTrend = trendChart(
        trend,
        aggregation as Aggregation,
        granularity as Granularity
    )
    const trendStacked = buildTrend(
        trend,
        aggregation as Aggregation,
        granularity as Granularity,
        5
    )

    const visualMap = generateVisualMap(theTrend.flag, theTrend.label)
    useEffect(() => {
        setSelectedDatapoint(undefined)
    }, [chartLayout, chartType, granularity, aggregation])

    return (
        <Card>
            <Grid numItems={6} className="gap-4 mb-4">
                <Col numColSpan={2}>
                    <AssetChartMetric
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
                    <AssetChartSelectors
                        timeRange={timeRange}
                        chartType={chartType as ChartType}
                        setChartType={setChartType}
                        granularity={granularity as Granularity}
                        setGranularity={(v) => {
                            setGranularity(v)
                            onGranularityChanged(v)
                        }}
                        chartLayout={chartLayout}
                        setChartLayout={setChartLayout}
                        validChartLayouts={validChartLayouts}
                        aggregation={aggregation as Aggregation}
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
                    chartType={chartType as ChartType}
                    loading={isLoading}
                    error={error}
                    isCost={false}
                />
            ) : (
                <Chart
                    labels={theTrend.label}
                    chartData={theTrend.data}
                    chartType={chartType as ChartType}
                    chartAggregation={
                        aggregation === 'trending' ? 'trend' : 'cumulative'
                    }
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
                            const t1 = trend
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
