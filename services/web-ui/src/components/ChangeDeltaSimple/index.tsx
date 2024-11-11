import { Color, DeltaType, Flex, Text, Icon } from '@tremor/react'
import { ChevronDownIcon } from '@heroicons/react/24/outline'
import { ArrowUpFill, ArrowDownFill, ArrowRightFill } from '../../icons/icons'
import { numberDisplay } from '../../utilities/numericDisplay'

interface IChangeDelta {
    change: string | number | undefined
    maxChange?: number
    isDelta?: boolean
    children?: any
}

const properties = (
    change: string | number | undefined,
    isDelta: boolean | undefined
) => {
    let color: Color = 'amber'
    let delta: DeltaType = 'unchanged'
    let icon: any
    if (Number(change) < 0) {
        color = 'rose'
        delta = 'decrease'
        icon = ArrowDownFill
    } else if (Number(change) > 0) {
        color = 'emerald'
        delta = 'increase'
        icon = ArrowUpFill
    } else if (Number(change) === 0) {
        color = 'gray'
        delta = 'unchanged'
        icon = ArrowRightFill
    }

    return {
        color,
        delta,
        icon,
    }
}

export default function BadgeDeltaSimple({
    change,
    maxChange,
    isDelta,
    children,
}: IChangeDelta) {
    const property = properties(change, isDelta)
    const ch = () => {
        const v = Math.abs(Number(change))
        if (maxChange !== undefined) {
            return Math.min(v, maxChange)
        }
        return v
    }
    const showPlus = () => {
        const v = Math.abs(Number(change))
        if (maxChange !== undefined) {
            return v > maxChange
        }
        return false
    }
    return (
        <Flex
            alignItems="center"
            justifyContent="start"
            className="gap-2 w-fit -ml-1"
        >
            <Flex className="w-fit" alignItems="center" justifyContent="start">
                <Icon
                    color={property.color}
                    icon={property.icon}
                    className="-mr-1"
                />
                <Text className="w-fit" color={property.color}>{`${
                    showPlus() ? '+' : ''
                }${numberDisplay(ch(), 0)} ${isDelta ? '' : '%'}`}</Text>
            </Flex>
            {children && <Text>{children}</Text>}
        </Flex>
    )
}
