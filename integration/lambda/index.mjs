export const handler = async (event) => {
    return {
        statusCode: 200,
        headers: {
            "Content-Type": "application/json",
        },
        body: JSON.stringify({
            message: "hello from lambda",
            method: event.httpMethod,
            path: event.path,
        }),
        isBase64Encoded: false,
    };
};
