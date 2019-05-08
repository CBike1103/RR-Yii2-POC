# RoadRunner + php7 + yii2 要点

## Memroy Usage
一方面要防止Memory Leak，一方面也要防止内存占用过多影响性能。

* 每次处理完请求调用`gc_collect_cycles()`方法
* 设定一个memory limit，超过上限worker会自动关闭。

代码大约如下
```php
public function clean($limit)
    {
        gc_collect_cycles();
        if (!isset($limit)) {
            $limit = $this->getMemoryLimit();
        } else {
            $limit = $limit * 1024;
        }
        $limit = $this->getMemoryLimit();
        $bound = $limit * .90;
        $usage = memory_get_usage(true);
        if ($usage >= $bound) {
            return true;
        }

        return false;
    }
```
## opcache
* 控制台脚本中的opcache开关与否对性能的影响有限。opcache.file_cache对控制台脚本性能有所提升。具体影响有待进一步验证。

## connections - TODO
* 在yii2 db设置中配置
    ```php
    $db['attributes'] = [\PDO::ATTR_PERSISTENT => true];
    ```
    是可以在php-fpm中实现长连接的。可能担心连接数问题，目前的项目基本都没有开启。
    在控制台中，这个配置会失效。
* 长时运行的脚本数据库链接断线重连问题 https://www.yiichina.com/topic/7296?sort=desc
* redis问题类似mysql

## http 相关
* json数据的判断和解析需要自己实现
* 跳转问题

## 文件流 - TODO
* 发送文件流的支持有问题，这个需求我们似乎没有？

## 原因尚不明 - TODO
* 当开启yii2 debug时，相比fpm，控制台脚本性能下降非常显著。表现为 spl_autoload 函数耗时严重。
* 控制台脚本中的`date_default_timezone_set()`函数需要耗时3毫秒左右。

## echo问题 - TODO
* 有的同学写的代码是直接echo的，没有用Yii2的Response Class, 因此也就不能直接被yii2-psr7-bridge支持。需要我们在调用handler的前后加上`ob_start()`和`ob_get_clean()`方法。这对速度的影响有待进一步验证。

## response问题
* 有的同学写了通用的response返回方法如下
```php
/**
 * @desc 通用返回ajax请求方法
 * @param $code
 * @param string $msg
 * @param array $data
 * @param int $header
 */
public static function _end($code, $msg='', $data=array(), $header = 1)
{
    $response = Yii::$app->getResponse();
    if($header ==1){
        $response->getHeaders()->add('Content-Type', 'application/json;charset=utf-8');
    }
    $code_cut = substr($code,-3);
    $msg = !empty($msg) ? $msg : Yii::t('tips',$code_cut);
    $response->content = json_encode(array(
        "code"=>$code,
        "msg"=>$msg,
        "data"=>$data,
    ));
    \Yii::$app->end();
}
```
可能需要改成这样
```php
/* @desc 通用返回ajax请求方法
 * @param $code
 * @param string $msg
 * @param array $data
 * @param int $header
 */
public static function _end($code, $msg='', $data=array(), $header = 1)
{
    $response = Yii::$app->getResponse();
    if($header ==1){
        $response->getHeaders()->add('Content-Type', 'application/json;charset=utf-8');
    }
    $code_cut = substr($code,-3);
    $msg = !empty($msg) ? $msg : Yii::t('tips',$code_cut);
    $response->content = json_encode(array(
        "code"=>$code,
        "msg"=>$msg,
        "data"=>$data,
    ));
    return;
}
```
* php中自定义的header似乎失效了。

## 运行一个复杂项目并验证 - TODO