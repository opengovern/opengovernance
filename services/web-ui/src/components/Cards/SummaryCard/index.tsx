import { useAtomValue } from 'jotai'
import { Button, Card, Flex, Metric, Text } from '@tremor/react'
import { ArrowPathIcon, ChevronRightIcon } from '@heroicons/react/24/outline'
import { useNavigate } from 'react-router-dom'
import Spinner from '../../Spinner'
import {
    numberDisplay,
    numericDisplay,
} from '../../../utilities/numericDisplay'
import ChangeDelta from '../../ChangeDelta'
import { searchAtom } from '../../../utilities/urlstate'
import { SourceType } from '../../../api/api'
import { getConnectorIcon } from '../ConnectorCard'

type IProps = {
    title: string
    connector?: SourceType
    metric: string | number | any | undefined
    metricPrev?: string | number | undefined
    secondLine?: string
    unit?: string
    url?: string
    onClick?: () => void
    loading?: boolean
    border?: boolean
    blueBorder?: boolean
    error?: string
    onRefresh?: () => void
    isExact?: boolean
    isPrice?: boolean
    isPercent?: boolean
    isString?: boolean
    isElement?: boolean
    blur?: boolean
    blurSecondLine?: boolean
    cardClickable?: boolean
}

export default function SummaryCard({
    title,
    connector,
    metric,
    metricPrev,
    secondLine,
    isString = false,
    isElement = false,
    unit,
    url,
    loading = false,
    border = true,
    blueBorder = false,
    error,
    onRefresh,
    isExact = false,
    isPrice = false,
    isPercent = false,
    onClick,
    blur,
    blurSecondLine,
    cardClickable,
}: IProps) {
    const navigate = useNavigate()
    const searchParams = useAtomValue(searchAtom)

    const metricValue = () => {
        if (isElement) {
            return metric
        }

        if (isString) {
            return (
                <Text
                    className={`${
                        blur === true ? 'blur-sm' : ''
                    } text-gray-800 truncate`}
                >
                    {metric}
                </Text>
            )
        }

        return (
            <Metric>
                {isExact
                    ? `${isPrice ? '$' : ''}${numberDisplay(
                          metric,
                          isExact && isPrice ? 2 : 0
                      )}${isPercent ? '%' : ''}`
                    : `${isPrice ? '$' : ''}${numericDisplay(metric)}${
                          isPercent ? '%' : ''
                      }`}
            </Metric>
        )
    }

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
            <Flex flexDirection="col" alignItems="start">
                <Flex
                    justifyContent="start"
                    alignItems="end"
                    className="gap-1 mb-1"
                >
                    {metricValue()}
                    {!!unit && <Text className="mb-0.5">{unit}</Text>}
                    {!!metricPrev && (
                        <Text className="mb-0.5">
                            from{' '}
                            {isExact
                                ? `${isPrice ? '$' : ''}${numberDisplay(
                                      metricPrev,
                                      isExact && isPrice ? 2 : 0
                                  )}${isPercent ? '%' : ''}`
                                : `${isPrice ? '$' : ''}${numericDisplay(
                                      metricPrev
                                  )}${isPercent ? '%' : ''}`}
                        </Text>
                    )}
                </Flex>
                {!!secondLine && (
                    <Text
                        className={`${
                            blurSecondLine === true ? 'blur-sm' : ''
                        } w-full text-start mb-0.5 truncate`}
                    >
                        {secondLine}
                    </Text>
                )}
                {!!metricPrev && (
                    <Flex className="mt-1">
                        <ChangeDelta
                            change={
                                ((Number(metric) - Number(metricPrev)) /
                                    Number(metricPrev)) *
                                100
                            }
                        />
                    </Flex>
                )}
            </Flex>
        )
    }

    return (
        <Card
            key={title}
            onClick={() =>
                // eslint-disable-next-line no-nested-ternary
                url
                    ? navigate(`${url}?${searchParams}`)
                    : onClick
                    ? onClick()
                    : null
            }
            className={`${border ? '' : 'ring-0 !shadow-transparent p-0'} ${
                cardClickable ? 'cursor-pointer' : ''
            } ${url ? 'cursor-pointer' : ''} ${
                blueBorder ? 'border-l-openg-500 border-l-2' : ''
            }`}
        >
            <Flex justifyContent="start" className="mb-1.5">
                {connector && getConnectorIcon(connector, '!h-1 mr-2')}
                <Text className="font-semibold">{title}</Text>
                {!border && (url || onClick) && (
                    <ChevronRightIcon className="ml-1 h-4 text-openg-500" />
                )}
            </Flex>
            {loading ? (
                <div className="w-fit">
                    <Spinner />
                </div>
            ) : (
                <Flex alignItems="baseline">
                    <Flex>{value()}</Flex>
                    {border && (
                        <div className="justify-self-end">
                            {(url || (onClick && !cardClickable)) && (
                                <Button
                                    size="xs"
                                    variant="light"
                                    icon={ChevronRightIcon}
                                    iconPosition="right"
                                >
                                    See more
                                </Button>
                            )}
                        </div>
                    )}
                </Flex>
            )}
        </Card>
    )
}
