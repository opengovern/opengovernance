import { BadgeDelta, Card, Color, Flex, Metric, Text } from '@tremor/react'
import dayjs from 'dayjs'
import { numberDisplay } from '../../../utilities/numericDisplay'
import { badgeTypeByDelta } from '../../../utilities/deltaType'
import { renderDateText } from '../../Layout/Header/DatePicker'

interface IAssetChartMetric {
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
    error: string | undefined
    comparedToNextLine?: boolean
}

export function AssetChartMetric({
    title,
    timeRange,
    timeRangePrev,
    total,
    totalPrev,
    isLoading,
    error,
    comparedToNextLine,
}: IAssetChartMetric) {
    const deltaColors = new Map<string, Color>()
    deltaColors.set('increase', 'emerald')
    deltaColors.set('moderateIncrease', 'emerald')
    deltaColors.set('unchanged', 'orange')
    deltaColors.set('moderateDecrease', 'rose')
    deltaColors.set('decrease', 'rose')

    const changeRate = (
        ((Number(total) - Number(totalPrev)) / Number(totalPrev)) *
        100
    ).toFixed(1)

    return (
        <Card
            key={title}
            className="w-fit ring-0 shadow-transparent dark:shadow-none border-0 p-0"
        >
            <Flex alignItems="baseline">
                <Flex flexDirection="col" className="gap-3" alignItems="start">
                    <Flex justifyContent="start" alignItems="end">
                        <Text className="text-gray-800 dark:text-gray-100 ">
                            {title}
                        </Text>
                        <Text className="pl-2 ml-2 border-l border-gray-100 dark:border-gray-800">
                            {renderDateText(timeRange.start, timeRange.end)}
                        </Text>
                    </Flex>
                    <Flex justifyContent="start" alignItems="end">
                        {isLoading || (error !== undefined && error !== '') ? (
                            <div
                                className={`${
                                    error !== undefined && error !== ''
                                        ? ''
                                        : 'animate-pulse'
                                } h-2 mt-6 w-24 bg-slate-200 dark:bg-slate-700 rounded`}
                            />
                        ) : (
                            <Metric>{numberDisplay(total, 0)}</Metric>
                        )}
                    </Flex>
                    <Flex
                        flexDirection={comparedToNextLine ? 'col' : 'row'}
                        alignItems={comparedToNextLine ? 'start' : 'center'}
                        className="cursor-default"
                    >
                        {isLoading || (error !== undefined && error !== '') ? (
                            <div
                                className={`${
                                    error !== undefined && error !== ''
                                        ? ''
                                        : 'animate-pulse'
                                } h-7 w-20 my-1 bg-slate-200 dark:bg-slate-700 rounded-full`}
                            />
                        ) : (
                            <BadgeDelta
                                deltaType={badgeTypeByDelta(totalPrev, total)}
                            >
                                <Text
                                    color={deltaColors.get(
                                        badgeTypeByDelta(totalPrev, total)
                                    )}
                                >
                                    {changeRate} %
                                </Text>
                            </BadgeDelta>
                        )}

                        <Text className={comparedToNextLine ? '' : 'ml-1.5'}>
                            compared to{' '}
                            {renderDateText(
                                timeRangePrev.start,
                                timeRangePrev.end
                            )}
                        </Text>
                    </Flex>
                </Flex>
            </Flex>
        </Card>
    )
}
