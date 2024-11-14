import { DeltaType } from '@tremor/react'
import ChangeDelta from '../components/ChangeDelta'

export const badgeTypeByDelta = (
    oldValue?: number | string,
    newValue?: number | string
): DeltaType => {
    const changes = (Number(newValue) || 0) - (Number(oldValue) || 0)
    let deltaType: DeltaType = 'unchanged'
    if (changes === 0) {
        return deltaType
    }
    if (changes > 0) {
        deltaType = 'moderateIncrease'
    } else {
        deltaType = 'moderateDecrease'
    }
    return deltaType
}

export const percentageByChange = (oldValue?: number, newValue?: number) => {
    const changes =
        (((newValue || 0) - (oldValue || 0)) / (newValue || 1)) * 100
    return changes.toFixed(2)
}

export const deltaChange = (oldValue?: number, newValue?: number) => {
    return (newValue || 0) - (oldValue || 0)
}

export const badgeDelta = (
    oldValue?: number,
    newValue?: number,
    isDelta?: boolean,
    valueInsideBadge = false
) => {
    return oldValue === 0 && newValue !== 0 ? (
        ''
    ) : (
        <ChangeDelta
            change={
                isDelta
                    ? deltaChange(oldValue, newValue)
                    : percentageByChange(oldValue, newValue)
            }
            isDelta={isDelta}
            valueInsideBadge={valueInsideBadge}
        />
    )
}
