import { BadgeDelta, Color, DeltaType, Flex, Metric, Text } from '@tremor/react'
import { numberDisplay } from '../../utilities/numericDisplay'

interface IChangeDelta {
    change: string | number | undefined
    isDelta?: boolean
    valueInsideBadge?: boolean
    size?: 'sm' | 'xl'
}

const properties = (
    change: string | number | undefined,
    isDelta: boolean | undefined
) => {
    let color: Color = 'amber'
    let delta: DeltaType = 'unchanged'
    if (isDelta) {
        if (Number(change) < 0) {
            color = 'rose'
            // delta = 'decrease'
            delta = 'moderateDecrease'
        }
        if (Number(change) > 0) {
            color = 'emerald'
            // delta = 'increase'
            delta = 'moderateIncrease'
        }
    } else {
        if (Number(change) < 0) {
            color = 'rose'
            // if (Math.abs(Number(change)) > 10) {
            //     delta = 'decrease'
            // } else {
            delta = 'moderateDecrease'
            // }
        }
        if (Number(change) > 0) {
            color = 'emerald'
            // if (Number(change) > 10) {
            //     delta = 'increase'
            // } else {
            delta = 'moderateIncrease'
            // }
        }
    }

    return {
        color,
        delta,
    }
}

export default function ChangeDelta({
    change,
    isDelta,
    valueInsideBadge = false,
    size = 'sm',
}: IChangeDelta) {
    return (
        <Flex className="w-fit min-w-fit gap-1.5 h-full">
            <BadgeDelta
                size={size === 'sm' ? 'sm' : 'md'}
                deltaType={properties(change, isDelta).delta}
            >
                {valueInsideBadge &&
                    `${numberDisplay(
                        Math.abs(Number(change)),
                        isDelta ? 0 : 2
                    )} ${isDelta ? '' : '%'}`}
            </BadgeDelta>
            {!valueInsideBadge &&
                (size === 'sm' ? (
                    <Text
                        color={properties(change, isDelta).color}
                    >{`${numberDisplay(
                        Math.abs(Number(change)),
                        isDelta ? 0 : 2
                    )} ${isDelta ? '' : '%'}`}</Text>
                ) : (
                    <Metric
                        className="!text-2xl"
                        color={properties(change, isDelta).color}
                    >
                        {`${numberDisplay(
                            Math.abs(Number(change)),
                            isDelta ? 0 : 2
                        )} ${isDelta ? '' : '%'}`}
                    </Metric>
                ))}
        </Flex>
    )
}
