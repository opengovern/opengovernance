import {
    Callout,
    Card,
    Col,
    Flex,
    Grid,
    Select,
    Tab,
    TabGroup,
    TabList,
    Text,
} from '@tremor/react'
import { ReactNode, useEffect, useState } from 'react'
import { Dayjs } from 'dayjs'
import { capitalizeFirstLetter } from '../../utilities/labelMaker'
import { checkGranularity, generateItems } from '../../utilities/dateComparator'
import { BarChartIcon, LineChartIcon } from '../../icons/icons'
import { dateDisplay } from '../../utilities/dateDisplay'
import { numberDisplay } from '../../utilities/numericDisplay'
import Chart from '../Chart'
import { generateVisualMap, resourceTrendChart } from '../../pages/Assets'

interface ITrends {
    activeTimeRange: { start: Dayjs; end: Dayjs }
    trend: any[] | undefined
    trendName: string
    labels: string[]
    chartData:
        | (string | number | undefined)[]
        | (
              | {
                    name: string
                    value: string
                    itemStyle?: undefined
                    label?: undefined
                }
              | {
                    value: number
                    name: string
                    itemStyle: { color: string; decal: { symbol: string } }
                    label: { show: boolean }
                }
          )[]
        | (
              | {
                    name: string
                    value: number | undefined
                    itemStyle?: undefined
                    label?: undefined
                }
              | {
                    value: number
                    name: string
                    itemStyle: { color: string; decal: { symbol: string } }
                    label: { show: boolean }
                }
          )[]
        | undefined
    loading: boolean
    firstKPI?: ReactNode
    secondKPI?: ReactNode
    thirdKPI?: ReactNode
    onGranularityChange?: (gran: 'monthly' | 'daily' | 'yearly') => void
    isPercent?: boolean
    isCost?: boolean
}

export default function Trends({
    labels,
    chartData,
    loading,
    firstKPI,
    secondKPI,
    thirdKPI,
    activeTimeRange,
    trend,
    trendName,
    onGranularityChange,
    isPercent,
    isCost,
}: ITrends) {
    const [selectedChart, setSelectedChart] = useState<'line' | 'bar'>('bar')
    const [selectedIndex, setSelectedIndex] = useState(0)
    const [selectedGranularity, setSelectedGranularity] = useState<
        'monthly' | 'daily' | 'yearly'
    >(
        checkGranularity(activeTimeRange.start, activeTimeRange.end).daily
            ? 'daily'
            : 'monthly'
    )
    useEffect(() => {
        setSelectedGranularity(
            checkGranularity(activeTimeRange.start, activeTimeRange.end).monthly
                ? 'monthly'
                : 'daily'
        )
    }, [activeTimeRange])
    useEffect(() => {
        if (onGranularityChange) {
            onGranularityChange(selectedGranularity)
        }
    }, [selectedGranularity])
    useEffect(() => {
        if (selectedIndex === 0) setSelectedChart('line')
        if (selectedIndex === 1) setSelectedChart('bar')
    }, [selectedIndex])
    useEffect(() => {
        if (trend && trend?.length < 2) {
            setSelectedIndex(1)
        }
    }, [trend])

    const [selectedDatapoint, setSelectedDatapoint] = useState<any>(undefined)

    return (
        <Card>
            <Grid numItems={5} className="gap-4">
                {firstKPI || <Col />}
                {secondKPI ? (
                    <div className="border-l border-l-gray-200 h-full pl-3">
                        {secondKPI}
                    </div>
                ) : (
                    <Col />
                )}
                {thirdKPI ? (
                    <div className="border-l border-l-gray-200 h-full pl-3">
                        {thirdKPI}
                    </div>
                ) : (
                    <Col />
                )}
                <Col numColSpan={2}>
                    <Flex justifyContent="end" className="gap-4">
                        {!!onGranularityChange &&
                            generateItems(
                                activeTimeRange.start,
                                activeTimeRange.end,
                                capitalizeFirstLetter(selectedGranularity),
                                selectedGranularity,
                                (v) => {
                                    // eslint-disable-next-line @typescript-eslint/ban-ts-comment
                                    // @ts-ignore
                                    setSelectedGranularity(v)
                                }
                            )}
                        <TabGroup
                            index={selectedIndex}
                            onIndexChange={setSelectedIndex}
                            className="w-fit rounded-lg"
                        >
                            <TabList variant="solid">
                                <Tab value="line">
                                    <LineChartIcon className="h-5" />
                                </Tab>
                                <Tab value="bar">
                                    <BarChartIcon className="h-5" />
                                </Tab>
                            </TabList>
                        </TabGroup>
                    </Flex>
                </Col>
            </Grid>
            {trend
                ?.filter(
                    (t) =>
                        selectedDatapoint?.color === '#E01D48' &&
                        dateDisplay(t.date || t.timestamp * 1000) ===
                            selectedDatapoint?.name
                )
                .map((t) => (
                    <Callout
                        color="rose"
                        title="Incomplete data"
                        className="w-fit mt-4"
                    >
                        {`Checked ${numberDisplay(
                            t.totalSuccessfulDescribedConnectionCount ||
                                t.totalConnectionCount -
                                    t.failedConnectionCount,
                            0
                        )} ${trendName.toLowerCase()} out of ${numberDisplay(
                            t.totalConnectionCount,
                            0
                        )} on ${dateDisplay(t.date || t.timestamp * 1000)}`}
                    </Callout>
                ))}
            <Flex justifyContent="end" className="mt-2 gap-2.5">
                <div className="h-2.5 w-2.5 rounded-full bg-openg-800" />
                <Text>{trendName}</Text>
            </Flex>
            <Chart
                labels={labels}
                chartData={chartData}
                chartType={selectedChart}
                chartAggregation="trend"
                loading={loading}
                isPercent={isPercent}
                isCost={isCost}
                visualMap={
                    generateVisualMap(
                        resourceTrendChart(trend, selectedGranularity).flag,
                        resourceTrendChart(trend, selectedGranularity).label
                    ).visualMap
                }
                markArea={
                    generateVisualMap(
                        resourceTrendChart(trend, selectedGranularity).flag,
                        resourceTrendChart(trend, selectedGranularity).label
                    ).markArea
                }
                onClick={(p) => setSelectedDatapoint(p)}
            />
        </Card>
    )
}
