import {
    Button,
    Card,
    Flex,
    Metric,
    Text,
    Color,
    BadgeDelta,
} from '@tremor/react'
import { ArrowPathIcon, ChevronRightIcon } from '@heroicons/react/24/outline'

import { useNavigate, useSearchParams } from 'react-router-dom'
import { log } from 'console'
import { useAtomValue } from 'jotai'
import Spinner from '../../Spinner'
import {
    numberDisplay,
    numericDisplay,
} from '../../../utilities/numericDisplay'
import { badgeTypeByDelta } from '../../../utilities/deltaType'
import { searchAtom } from '../../../utilities/urlstate'

type IProps = {
    title: string
    metric: string | number | undefined
    metricPrev?: string | number | undefined
    unit?: string
    url?: string
    loading?: boolean
    prevLoading?: boolean
    border?: boolean
    blueBorder?: boolean
    error?: string
    onRefresh?: () => void
    isExact?: boolean
    isPrice?: boolean
    isPercent?: boolean
    isString?: boolean
}

export default function MetricCard({
    title,
    metric,
    metricPrev,
    isString = false,
    unit,
    url,
    loading = false,
    prevLoading = false,
    border = true,
    blueBorder = false,
    error,
    onRefresh,
    isExact = false,
    isPrice = false,
    isPercent = false,
}: IProps) {
    const navigate = useNavigate()
    const searchParams = useAtomValue(searchAtom)

    const changeRate = (
        ((Number(metric) - Number(metricPrev)) / Number(metricPrev)) *
        100
    ).toFixed(1)

    const deltaColors = new Map<string, Color>()
    deltaColors.set('increase', 'emerald')
    deltaColors.set('moderateIncrease', 'emerald')
    deltaColors.set('unchanged', 'orange')
    deltaColors.set('moderateDecrease', 'rose')
    deltaColors.set('decrease', 'rose')

    const value = () => {
        if (error !== undefined && error.length > 0) {
            return (
                <Flex
                    justifyContent="start"
                    alignItems="start"
                    className="cursor-pointer w-full"
                    onClick={onRefresh}
                >
                    <Text className="text-gray-400 mr-2 w-auto">
                        Error loading
                    </Text>
                    <Flex
                        flexDirection="row"
                        justifyContent="end"
                        className="w-auto"
                    >
                        <ArrowPathIcon className="text-blue-500 w-4 h-4 mr-1" />
                        <Text className="text-blue-500">Reload</Text>
                    </Flex>
                </Flex>
            )
        }
        return (
            <Flex flexDirection="col" className="gap-3" alignItems="start">
                {url && (
                    <Button
                        size="xs"
                        variant="light"
                        icon={ChevronRightIcon}
                        iconPosition="right"
                    >
                        {title}
                    </Button>
                )}

                <Flex justifyContent="start" alignItems="end">
                    {isString ? (
                        <Text className="text-gray-800">{metric}</Text>
                    ) : (
                        <Metric>
                            {isExact
                                ? `${isPrice ? '$' : ''}${numberDisplay(
                                      metric,
                                      0
                                  )}${isPercent ? '%' : ''}`
                                : `${isPrice ? '$' : ''}${numericDisplay(
                                      metric || 0
                                  )}${isPercent ? '%' : ''}`}
                        </Metric>
                    )}
                    {!!unit && <Text className="mb-0.5">{unit}</Text>}
                </Flex>

                {!!metricPrev && (
                    <Flex
                        flexDirection="row"
                        alignItems="center"
                        className="cursor-default"
                    >
                        <BadgeDelta
                            deltaType={badgeTypeByDelta(metricPrev, metric)}
                        />

                        <Text
                            color={deltaColors.get(
                                badgeTypeByDelta(metricPrev, metric)
                            )}
                            className="ml-2"
                        >
                            {changeRate} %
                        </Text>

                        <Text className="ml-1.5">
                            compared to previous period
                        </Text>
                    </Flex>
                )}
            </Flex>
        )
    }

    return (
        <Card
            key={title}
            onClick={() => (url ? navigate(`${url}?${searchParams}`) : null)}
            className={`w-fit ${
                border ? '' : 'ring-0 shadow-transparent p-0'
            } ${url ? 'cursor-pointer' : ''} ${
                blueBorder ? 'border-l-openg-500 border-l-2' : ''
            }`}
        >
            {loading ? (
                <div className="w-fit">
                    <Spinner />
                </div>
            ) : (
                <Flex alignItems="baseline">
                    <Flex>{value()}</Flex>
                </Flex>
            )}
        </Card>
    )
}
