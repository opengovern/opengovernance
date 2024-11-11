import { Button, Flex, Grid, Subtitle, Title } from '@tremor/react'
import stackImg from '../../icons/stack.png'
import TopHeader from '../../components/Layout/Header'

export default function Stack() {
    return (
        <>
            <TopHeader />
            <Flex justifyContent="center" className="md:mt-32">
                <Grid numItems={1} numItemsMd={2}>
                    <img src={stackImg} alt="stack" loading="lazy" />
                    <Flex
                        flexDirection="col"
                        alignItems="start"
                        justifyContent="center"
                        className="p-12"
                    >
                        <Title className="font-semibold">
                            Under Construction
                        </Title>
                        <Subtitle className="mt-3 mb-6 text-gray-600">
                            We are working on it, check out the CLI for working
                            stacks
                        </Subtitle>
                        <Button
                            onClick={() =>
                                // eslint-disable-next-line @typescript-eslint/ban-ts-comment
                                // @ts-ignore
                                document.getElementById('CLI').click()
                            }
                        >
                            Go to CLI
                        </Button>
                    </Flex>
                </Grid>
            </Flex>
        </>
    )
}
