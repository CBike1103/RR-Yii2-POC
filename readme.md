# 用RoadRunner运行Yii2项目

## 1. 优势
将yii2框架改为长时运行的脚本程序，作为RoadRunner的worker. 可以节省PHP进程重启所消耗的时间。

实际测试运行效率提高相比 nginx+php7+fpm 提高约50%
## 2. 原理

1. RoadRunner原理

    参考[这里](https://zhuanlan.zhihu.com/p/60599237)。
    简单来说就是用长时运行的php脚本作为worker来处理http请求。
2. roadrunner与yii2

    基于roadrunner官方repo的[讨论](https://github.com/spiral/roadrunner/issues/78)，我们对 https://github.com/charlesportwoodii/yii2-psr7-bridge 这个库进行了改良。使得yii2项目在代码几乎不用更改的情况下可以作为RoadRunner。

## 3. 具体用法
1. 在项目中引入yii2-psr7-bridge，
    ```
    composer require yoozoo/yii2-psr7-bridge
    ```
2. 更改（或者创建一个新的）配置文件，将`request`和`response`的class改为`yii\Psr7\web\Request`和`yii\Psr7\web\Response`，例如:

    ```php
    return [
        'components' => [
            'request' => [
                'class' => \yii\Psr7\web\Request::class,
            ],
            'response' => [
                'class' => \yii\Psr7\web\Response::class
            ],
        ]
    ];
    ```


3. 添加一个脚本入口文件，内容大致如下

    ```php
    #!/usr/bin/env php
    <?php
    // 错误信息一定要输出到stderr，这样才能被RoadRunner捕捉到
    ini_set('display_errors', 'stderr');

    defined('YII_DEBUG') or define('YII_DEBUG', true);
    defined('YII_ENV') or define('YII_ENV', 'dev');

    require_once '/path/to/vendor/autoload.php';
    require_once '/path/to/vendor/yiisoft/yii2/Yii.php';
    $config = require_once '/path/to/config/config.php';

    // Roadrunner relay and PSR7 object
    $relay = new \Spiral\Goridge\StreamRelay(STDIN, STDOUT);
    $psr7 = new \Spiral\RoadRunner\PSR7Client(new \Spiral\RoadRunner\Worker($relay));

    $application = (new \yii\Psr7\web\Application($config));

    // Handle each request in a loop
    while ($request = $psr7->acceptRequest()) {
        try {
            $response = $application->handle($request);
            $psr7->respond($response);

            /*
            // 如果程序中使用了echo或者var_dump等, 需要做如下改进
            // roadrunner需要接收psr7标准的Response
            // 直接输出到stdout是RoadRunner不允许的

            ob_start();
            $response = $application->handle($request);
            $echoResponse = ob_get_clean();
            if (empty($echoResponse)) {
                $psr7->respond($response);
            } else {
                $resp = new \Zend\Diactoros\Response();
                $resp->getBody()->write($echoResponse);
                $psr7->respond($resp);
            }*/

        } catch (\Throwable $e) {
            $psr7->getWorker()->error((string)$e);
        }

        if ($application->clean()) {
            $psr7->getWorker()->stop();
            return;
        }
    }
    ```

4. 编辑RoadRunner的配置文件，添加wroker运行时的环境变量

    ```yaml
    env:
        YII_ALIAS_WEBROOT: /path/to/webroot
        YII_ALIAS_WEB: '127.0.0.1:8080'
    ```

    > 这些环境变量是**必须**的
## 4. 我们做的改进
1. 使yii2支持psr7标准

    我们创建了新的`\yii\Psr7\web\Request`和`\yii\Psr7\web\Response`类，分别继承了Yii2框架中原有的类，并实现了psr7中定义的接口。使得原有的yii2程序可以在不进行大的改动的基础上支持psr7.
2. 使yii2程序可以长时运行

    我们的创建了新的`\yii\Psr7\web\Application`类，作为脚本程序的主体。每次接受完请求之后，Application类不会退出，而是调用`clean()`方法进行垃圾回收。该方法可以传入一个参数，定义内存占用的上限。如果该进程内存占用超过上限，就会通知RoadRunner结束进程。

    我们同时重写了原本Application类中的`_end()`方法，确保它只是返回本次请求的结果而不会彻底退出。
    > 但是我们对手动调用`die()`或者`exit()`的同学还是没有办法。好在RoadRunner会在脚本退出之后立刻开启新的Worker，虽然还是会对性能产生影响。

    每次处理完请求后，Application会遍历所有的Component，并关闭其中所有实现了Connection的Component. 在之后的版本中，会加入对连接池的管理，以进一步提高性能和稳定性。

3. 对应脚本程序做的改进

    我们在Application每次处理请求之前，会先创建 _SERVER, _POST等全球变量，以便程序可以和在fpm中运行时的行为一致。