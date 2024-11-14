import { ITrendItem } from '.'
import { dateDisplay } from '../../../utilities/dateDisplay'
import { StackItem } from '../../Chart/Stacked'

export const trendChart = (trend: ITrendItem[] | undefined) => {
    const label: string[] = []
    const data: StackItem[][] = []
    const colors: string[] = []

    const colorMapping: { [severity: string]: string } = {
        Critical: '#6E120B',
        High: '#CA2B1D',
        Medium: '#EE9235',
        Low: '#F4C744',
        black: '#000',
        None: '#9BA2AE',
        Passed: '#54B584',
        Failed: '#CA2B1D',
        Score: '#F4C744',
    }

    if (!trend) {
        return {
            label,
            data,
            colors,
        }
    }

    for (let i = 0; i < trend?.length; i += 1) {
        label.push(dateDisplay(trend[i]?.timestamp))
        const stackData: StackItem[] = []

        for (let j = 0; j < trend[i].stack.length; j += 1) {
            stackData.push({
                value: Number(trend[i]?.stack[j].count || 0),
                label: String(trend[i]?.stack[j].name),
            })
        }
        data.push(stackData)
    }

    const p = trend
        .flatMap((v) => v.stack)
        .map((v) => v.name)
        .reduce<string[]>((prev, current) => {
            return prev.includes(current) ? prev : [...prev, current]
        }, [])
        .map((lbl) => {
            return colorMapping[lbl]
        })

    return {
        label,
        data,
        colors: p,
    }
}
