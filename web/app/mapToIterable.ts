/**
 * Created by erdemaslan on 27/04/16.
 */
import {Pipe, PipeTransform} from 'angular2/core';
/**
 * Map to Iteratble Pipe
 *
 * It accepts Objects and [Maps](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Map)
 *
 * Example:
 *
 *  <div *ngFor="#keyValuePair of someObject | mapToIterable">
 *    key {{keyValuePair.key}} and value {{keyValuePair.value}}
 *  </div>
 *
 */
@Pipe({ name: 'mapToIterable' })
export class MapToIterable implements PipeTransform {
    transform(value) {
        let result = [];

        if(value.entries) {
            for (var [key, value] of value.entries()) {
                result.push({ key, value });
            }
        } else {
            for(let key in value) {
                result.push({ key, value: value[key] });
            }
        }

        return result;
    }
}
